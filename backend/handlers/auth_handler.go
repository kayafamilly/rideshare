package handlers

import (
	"errors" // Import errors package for errors.As
	"fmt"    // For string formatting
	"log"    // For logging
	"net/http"

	"github.com/go-playground/validator/v10" // Import validator for error checking
	"github.com/gofiber/fiber/v2"            // Fiber framework
	"github.com/google/uuid"

	"rideshare/backend/models"   // Local models package
	"rideshare/backend/services" // Local services package
)

// AuthHandler handles HTTP requests related to authentication.
type AuthHandler struct {
	authService *services.AuthService // Instance of AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignUp handles the POST /api/v1/auth/signup request.
func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	// 1. Parse request body into SignUpRequest struct
	var req models.SignUpRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing signup request body: %v", err)
		// Return bad request error if parsing fails
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"details": err.Error(),
		})
	}

	// Log the received request (excluding password for security)
	log.Printf("Received signup request for email: %s", req.Email)

	// 2. Call the AuthService to handle signup logic
	// Use context from Fiber context
	user, err := h.authService.SignUp(c.Context(), req)
	if err != nil {
		log.Printf("Error during signup process for email %s: %v", req.Email, err)
		// Determine appropriate HTTP status code based on error type
		statusCode := fiber.StatusInternalServerError
		errorMessage := "Signup failed due to an internal error"
		// Check for specific user-friendly errors from the service
		if err.Error() == "email or WhatsApp number already registered" {
			statusCode = fiber.StatusConflict // 409 Conflict
			errorMessage = err.Error()
		} else if err.Error() == "invalid birth date format (use YYYY-MM-DD)" {
			statusCode = fiber.StatusBadRequest // 400 Bad Request
			errorMessage = err.Error()
		} else {
			// Check if the underlying error is a validation error
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = fiber.StatusBadRequest // 400 Bad Request
				// Provide a more specific validation error message
				// You can customize this further based on validationErrors if needed
				errorMessage = fmt.Sprintf("Invalid signup data: %v", validationErrors)
			}
			// If it's not one of the specific errors or a validation error, it remains 500
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// Optionally include more details in non-production environments
			// "details": err.Error(),
		})
	}

	// 3. Return successful response (user data without sensitive info)
	log.Printf("Signup successful for user: %s (ID: %s)", user.Email, user.ID)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "User registered successfully",
		"data":    user, // User object (PasswordHash is already excluded in the model)
	})
}

// Login handles the POST /api/v1/auth/login request.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	// 1. Parse request body into LoginRequest struct
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing login request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"details": err.Error(),
		})
	}

	// Log the received request
	log.Printf("Received login request for email: %s", req.Email)

	// 2. Call the AuthService to handle login logic
	loginResponse, err := h.authService.Login(c.Context(), req)
	if err != nil {
		log.Printf("Error during login process for email %s: %v", req.Email, err)
		statusCode := fiber.StatusInternalServerError
		errorMessage := "Login failed due to an internal error"
		// Check for specific user-friendly errors
		if err.Error() == "invalid email or password" {
			statusCode = fiber.StatusUnauthorized // 401 Unauthorized
			errorMessage = err.Error()
		} else {
			// Check if the underlying error is a validation error
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = fiber.StatusBadRequest // 400 Bad Request
				errorMessage = fmt.Sprintf("Invalid login data: %v", validationErrors)
			}
			// If it's not one of the specific errors or a validation error, it remains 500
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// "details": err.Error(), // Optional details
		})
	}

	// 3. Return successful response with token and user data
	log.Printf("Login successful for user: %s (ID: %s)", loginResponse.User.Email, loginResponse.User.ID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Login successful",
		"data":    loginResponse, // Contains token and user info
	})
}

// UpdateProfile handles PUT /api/v1/users/profile
// Requires authentication.
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (UpdateProfile)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Missing user identification."})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Invalid user identification format."})
		}
		userID = parsedID
	}

	// 2. Parse request body into UpdateProfileRequest struct
	var req models.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing update profile request body for user %s: %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"details": err.Error(),
		})
	}

	log.Printf("Received update profile request from user %s: %+v", userID, req)

	// 3. Call service to update profile
	updatedUser, err := h.authService.UpdateProfile(c.Context(), userID, req)
	if err != nil {
		log.Printf("Error updating profile for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to update profile"

		errMsg := err.Error()
		if errMsg == "no update data provided" || errMsg == "invalid birth date format (use YYYY-MM-DD)" {
			statusCode = http.StatusBadRequest
			errorMessage = errMsg
		} else if errMsg == "whatsapp number already registered" {
			statusCode = http.StatusConflict
			errorMessage = errMsg
		} else if errMsg == "user not found or deleted" {
			statusCode = http.StatusNotFound // Or Forbidden? NotFound seems appropriate
			errorMessage = errMsg
		} else {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = http.StatusBadRequest
				errorMessage = fmt.Sprintf("Invalid profile data: %v", validationErrors)
			}
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
		})
	}

	// 4. Return successful response
	log.Printf("Profile updated successfully for user %s", userID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Profile updated successfully",
		"data":    updatedUser, // Return updated user data (without password/deleted_at)
	})
}

// DeleteAccount handles DELETE /api/v1/users/account
// Requires authentication.
func (h *AuthHandler) DeleteAccount(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (DeleteAccount)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Missing user identification."})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Invalid user identification format."})
		}
		userID = parsedID
	}

	log.Printf("Received delete account request from user %s", userID)

	// 2. Call service to perform soft delete
	err := h.authService.DeleteAccount(c.Context(), userID)
	if err != nil {
		log.Printf("Error deleting account for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to delete account"

		if err.Error() == "user not found or already deleted" {
			statusCode = http.StatusNotFound
			errorMessage = err.Error()
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
		})
	}

	// 3. Return successful response
	log.Printf("Account deleted successfully for user %s", userID)
	// Use 200 OK or 204 No Content for successful deletion
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Account deleted successfully",
	})
}

// SetupAuthRoutes registers the authentication routes with the Fiber app group.
// SetupUserRoutes registers user profile and account management routes.
// Requires authentication middleware.
func SetupUserRoutes(api fiber.Router, authService *services.AuthService, authMiddleware fiber.Handler) {
	handler := NewAuthHandler(authService) // Reuse AuthHandler for these related actions

	userGroup := api.Group("/users") // Create /api/v1/users group
	userGroup.Put("/profile", authMiddleware, handler.UpdateProfile)
	userGroup.Delete("/account", authMiddleware, handler.DeleteAccount)
	// Add GET /profile later if needed

	log.Println("User routes (/users/profile, /users/account) setup complete.")
}

// SetupAuthRoutes registers the public authentication routes.
func SetupAuthRoutes(api fiber.Router, authService *services.AuthService) {
	handler := NewAuthHandler(authService)

	authGroup := api.Group("/auth") // Create /api/v1/auth group
	authGroup.Post("/signup", handler.SignUp)
	authGroup.Post("/login", handler.Login)

	log.Println("Authentication routes (/auth/signup, /auth/login) setup complete.")
}
