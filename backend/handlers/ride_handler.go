package handlers

import (
	"fmt" // Import fmt for error formatting
	"log"
	"net/http" // For status codes

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"rideshare/backend/models"
	"rideshare/backend/services"
	// Import middleware package once created, e.g., "rideshare/backend/middleware"
)

// RideHandler handles HTTP requests related to rides.
type RideHandler struct {
	rideService *services.RideService
	// authService *services.AuthService // Might be needed if we fetch creator details here
}

// NewRideHandler creates a new RideHandler instance.
func NewRideHandler(rideService *services.RideService) *RideHandler {
	return &RideHandler{
		rideService: rideService,
	}
}

// CreateRide handles POST /api/v1/rides
// Requires authentication.
func (h *RideHandler) CreateRide(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context (set by auth middleware)
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		// Attempt to retrieve as string and parse, as some middleware might store it as string
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context or invalid type in CreateRide")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Missing user identification.",
			})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Invalid user identification format.",
			})
		}
		userID = parsedID // Assign the parsed UUID
	}

	// 2. Parse request body
	var req models.CreateRideRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing create ride request body for user %s: %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"details": err.Error(),
		})
	}

	// Log request
	log.Printf("Received create ride request from user %s: %+v", userID, req)

	// 3. Call service to create ride
	ride, err := h.rideService.CreateRide(c.Context(), req, userID)
	if err != nil {
		log.Printf("Error creating ride for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to create ride due to an internal error"

		// Handle specific errors from service
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			statusCode = http.StatusBadRequest
			errorMessage = fmt.Sprintf("Invalid ride data: %v", validationErrors)
		} else if err.Error() == "departure date and time must be in the future" {
			statusCode = http.StatusBadRequest
			errorMessage = err.Error()
		} else if err.Error() == "invalid departure date format (use YYYY-MM-DD)" || err.Error() == "invalid departure date or time format" || err.Error() == "departure or arrival coordinates are missing" {
			// Added check for missing coordinates error from service
			statusCode = http.StatusBadRequest
			errorMessage = err.Error()
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// "details": err.Error(), // Optional
		})
	}

	// 4. Return successful response
	log.Printf("Ride created successfully (ID: %s) by user %s", ride.ID, userID)
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Ride created successfully",
		"data":    ride,
	})
}

// ListAvailableRides handles GET /api/v1/rides
// Publicly accessible (no auth required).
func (h *RideHandler) ListAvailableRides(c *fiber.Ctx) error {
	log.Println("Received request to list available rides")
	// TODO: Add query param handling for filtering/pagination

	rides, err := h.rideService.ListAvailableRides(c.Context())
	if err != nil {
		log.Printf("Error listing available rides: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to retrieve available rides",
			// "details": err.Error(), // Optional
		})
	}

	log.Printf("Returning %d available rides", len(rides))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Available rides retrieved successfully",
		"data":    rides,
	})
}

// GetRideDetails handles GET /api/v1/rides/{id}
// Requires authentication.
func (h *RideHandler) GetRideDetails(c *fiber.Ctx) error {
	// 1. Get ride ID from URL parameter
	rideIDParam := c.Params("id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid ride ID format",
		})
	}

	// Optional: Get user ID from context if needed for authorization checks later
	// userID, _ := c.Locals("userID").(uuid.UUID)

	log.Printf("Received request for ride details: ID %s", rideID)

	// 2. Call service to get ride details
	ride, err := h.rideService.GetRideDetails(c.Context(), rideID)
	if err != nil {
		log.Printf("Error getting ride details for ID %s: %v", rideID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to retrieve ride details"
		if err.Error() == "ride not found" {
			statusCode = http.StatusNotFound
			errorMessage = err.Error()
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// "details": err.Error(), // Optional
		})
	}

	// 3. Return successful response
	log.Printf("Returning details for ride ID %s", rideID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Ride details retrieved successfully",
		"data":    ride,
	})
}

// JoinRide handles POST /api/v1/rides/{id}/join
// Requires authentication.
func (h *RideHandler) JoinRide(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		// Attempt to retrieve as string and parse, as some middleware might store it as string
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context or invalid type in JoinRide")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Missing user identification.",
			})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Invalid user identification format.",
			})
		}
		userID = parsedID // Assign the parsed UUID
	}

	// 2. Get ride ID from URL parameter
	rideIDParam := c.Params("id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for join request: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid ride ID format",
		})
	}

	log.Printf("Received request from user %s to join ride %s", userID, rideID)

	// 3. Call service to handle joining the ride
	participant, err := h.rideService.JoinRide(c.Context(), rideID, userID)
	if err != nil {
		log.Printf("Error joining ride %s for user %s: %v", rideID, userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to join ride due to an internal error"

		// Handle specific errors from service
		errMsg := err.Error()
		switch errMsg {
		case "ride not found":
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		case "ride is not open for joining", "ride is already full", "you cannot join your own ride", "you have already joined this ride":
			statusCode = http.StatusConflict // 409 Conflict for business rule violations
			errorMessage = errMsg
		case "database does not support transactions required for JoinRide":
			// This indicates a setup issue, likely internal error
			errorMessage = "Cannot process join request at this time."
		default:
			// Keep internal server error for other db errors
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// "details": err.Error(), // Optional
		})
	}

	// 4. Return successful response (participant details)
	log.Printf("User %s joined ride %s successfully (Participant ID: %s)", userID, rideID, participant.ID)
	// Create the specific response structure
	response := models.JoinRideResponse{
		ParticipationID: participant.ID,
		RideID:          participant.RideID,
		UserID:          participant.UserID,
		Status:          participant.Status,
		Message:         "Successfully joined ride. Proceed to payment.", // Or similar message
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{ // 200 OK might be better than 201 Created here
		"status":  "success",
		"message": response.Message,
		"data":    response,
	})
}

// GetRideContacts handles GET /api/v1/rides/{id}/contacts
// Requires authentication, user must be a confirmed participant or creator.
func (h *RideHandler) GetRideContacts(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	requestingUserID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (GetRideContacts)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Missing user identification."})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Invalid user identification format."})
		}
		requestingUserID = parsedID
	}

	// 2. Get ride ID from URL parameter
	rideIDParam := c.Params("id") // Use "id" to match route definition
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for get contacts: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	log.Printf("Received request from user %s to get contacts for ride %s", requestingUserID, rideID)

	// 3. Call service to get contacts (service handles authorization check)
	contacts, err := h.rideService.GetRideContacts(c.Context(), rideID, requestingUserID)
	if err != nil {
		log.Printf("Error getting contacts for ride %s, requested by user %s: %v", rideID, requestingUserID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to retrieve ride contacts"

		errMsg := err.Error()
		switch errMsg {
		case "ride not found":
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		case "unauthorized to view contacts for this ride":
			statusCode = http.StatusForbidden // 403 Forbidden is more appropriate than 401 here
			errorMessage = errMsg
		default:
			// Keep internal server error for other db errors
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
		})
	}

	// 4. Return successful response
	log.Printf("Returning %d contacts for ride %s to user %s", len(contacts), rideID, requestingUserID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Ride contacts retrieved successfully",
		"data":    contacts,
	})
}

// SearchRides handles GET /api/v1/rides/search
// Publicly accessible. Parses query parameters.
func (h *RideHandler) SearchRides(c *fiber.Ctx) error {
	// Parse query parameters into SearchRidesRequest struct
	var params models.SearchRidesRequest
	if err := c.QueryParser(&params); err != nil {
		log.Printf("Error parsing search query parameters: %v", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid search query parameters",
			"details": err.Error(),
		})
	}

	// Optional: Validate parsed parameters if needed (e.g., date format)
	// The service layer might also perform validation.

	log.Printf("Received ride search request with params: %+v", params)

	// Call service to search rides
	rides, err := h.rideService.SearchRides(c.Context(), params)
	if err != nil {
		log.Printf("Error searching rides with params %+v: %v", params, err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to search for rides",
		})
	}

	log.Printf("Returning %d rides for search params %+v", len(rides), params)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Rides search successful",
		"data":    rides,
	})
}

// ListUserCreatedRides handles GET /api/v1/users/me/rides/created
// Requires authentication.
func (h *RideHandler) ListUserCreatedRides(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok { /* ... handle missing/invalid userID ... */
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid ID"})
		}
		userID = parsedID
	}

	log.Printf("Received request for rides created by user %s", userID)
	rides, err := h.rideService.ListUserCreatedRides(c.Context(), userID)
	if err != nil { /* ... handle error ... */
		log.Printf("Error fetching created rides for user %s: %v", userID, err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failed to retrieve created rides"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": rides})
}

// ListUserJoinedRides handles GET /api/v1/users/me/rides/joined
// Requires authentication.
func (h *RideHandler) ListUserJoinedRides(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok { /* ... handle missing/invalid userID ... */
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid ID"})
		}
		userID = parsedID
	}

	log.Printf("Received request for rides joined by user %s", userID)
	rides, err := h.rideService.ListUserJoinedRides(c.Context(), userID)
	if err != nil { /* ... handle error ... */
		log.Printf("Error fetching joined rides for user %s: %v", userID, err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failed to retrieve joined rides"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": rides})
}

// ListUserHistoryRides handles GET /api/v1/users/me/rides/history
// Requires authentication.
func (h *RideHandler) ListUserHistoryRides(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok { /* ... handle missing/invalid userID ... */
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid ID"})
		}
		userID = parsedID
	}

	log.Printf("Received request for ride history for user %s", userID)
	rides, err := h.rideService.ListUserHistoryRides(c.Context(), userID)
	if err != nil { /* ... handle error ... */
		log.Printf("Error fetching history rides for user %s: %v", userID, err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failed to retrieve ride history"})
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": rides})
}

// DeleteRide handles DELETE /api/v1/rides/{id}
// Requires authentication.
func (h *RideHandler) DeleteRide(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok { /* ... handle missing/invalid userID ... */
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid ID"})
		}
		userID = parsedID
	}
	rideIDParam := c.Params("id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil { /* ... handle invalid ride ID ... */
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	log.Printf("Received delete request for ride %s from user %s", rideID, userID)
	hadParticipants, err := h.rideService.DeleteRide(c.Context(), rideID, userID)
	if err != nil { /* ... handle service error (not found, unauthorized, db error) ... */
		statusCode := http.StatusInternalServerError
		message := "Failed to delete ride"
		errMsg := err.Error()
		if errMsg == "ride not found" {
			statusCode = http.StatusNotFound
			message = errMsg
		} else if errMsg == "unauthorized to delete this ride" {
			statusCode = http.StatusForbidden
			message = errMsg
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": message})
	}

	message := "Ride deleted successfully."
	if hadParticipants {
		message = "Ride cancelled successfully. Participants were notified (logic pending)." // Adjust message
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": message})
}

// LeaveRide handles POST /api/v1/rides/{id}/leave
// Requires authentication.
func (h *RideHandler) LeaveRide(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (LeaveRide)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized"})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Invalid ID"})
		}
		userID = parsedID
	}
	rideIDParam := c.Params("id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for leave request: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	log.Printf("Received leave request for ride %s from user %s", rideID, userID)
	err = h.rideService.LeaveRide(c.Context(), rideID, userID)
	if err != nil {
		log.Printf("Error leaving ride %s for user %s: %v", rideID, userID, err)
		statusCode := http.StatusInternalServerError
		message := "Failed to leave ride"
		errMsg := err.Error()
		if errMsg == "could not leave ride: participation not found or already left/cancelled" {
			statusCode = http.StatusConflict
			message = errMsg
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": message})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "message": "Successfully left the ride."})
}

// GetMyParticipationStatus handles GET /api/v1/rides/{id}/my-status
// Requires authentication.
func (h *RideHandler) GetMyParticipationStatus(c *fiber.Ctx) error {
	// 1. Get authenticated user ID
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (GetMyParticipationStatus)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Missing user identification."})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Invalid user identification format."})
		}
		userID = parsedID
	}

	// 2. Get ride ID from URL parameter
	rideIDParam := c.Params("id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for status request: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	log.Printf("Received request for participation status for user %s on ride %s", userID, rideID)

	// 3. Call service to get status
	status, err := h.rideService.GetUserParticipationStatus(c.Context(), rideID, userID)
	if err != nil {
		log.Printf("Error fetching participation status for user %s, ride %s: %v", userID, rideID, err)
		// Don't expose internal DB errors directly
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"status": "error", "message": "Failed to retrieve participation status"})
	}

	// 4. Return status
	log.Printf("Returning participation status '%s' for user %s on ride %s", status, userID, rideID)
	return c.Status(http.StatusOK).JSON(fiber.Map{"status": "success", "data": fiber.Map{"participation_status": status}})
}

// SetupRideRoutes registers the ride-related routes with the Fiber app group.
// It requires the auth middleware for protected routes.
func SetupRideRoutes(api fiber.Router, rideService *services.RideService, authMiddleware fiber.Handler) {
	handler := NewRideHandler(rideService)

	// Public routes
	api.Get("/rides/search", handler.SearchRides) // New search endpoint
	api.Get("/rides", handler.ListAvailableRides) // Keep old endpoint for all available? Or remove? Let's keep for now.

	// Protected routes
	rideGroup := api.Group("/rides", authMiddleware) // Apply middleware to group for protected routes
	rideGroup.Post("/", handler.CreateRide)
	rideGroup.Get("/:id", handler.GetRideDetails)
	rideGroup.Post("/:id/join", handler.JoinRide)
	rideGroup.Get("/:id/contacts", handler.GetRideContacts)
	rideGroup.Delete("/:id", handler.DeleteRide)    // New delete route
	rideGroup.Post("/:id/leave", handler.LeaveRide) // New leave route

	// Routes for user-specific rides (My Rides) - Protected
	userRideGroup := api.Group("/users/me/rides", authMiddleware)
	userRideGroup.Get("/created", handler.ListUserCreatedRides)
	userRideGroup.Get("/joined", handler.ListUserJoinedRides)
	userRideGroup.Get("/history", handler.ListUserHistoryRides) // Add history route
	// Add route for participation status under rides group
	rideGroup.Get("/:id/my-status", handler.GetMyParticipationStatus)

	log.Println("Ride related routes setup complete.")
}
