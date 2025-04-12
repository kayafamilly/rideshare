package handlers

import (
	"fmt"      // Import fmt
	"log"      // For error checking
	"net/http" // For status codes and request object

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"rideshare/backend/services" // Local services
)

// PaymentHandler handles HTTP requests related to payments.
type PaymentHandler struct {
	paymentService *services.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler instance.
func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePaymentIntent handles POST /api/v1/rides/:ride_id/create-payment-intent
// Requires authentication.
func (h *PaymentHandler) CreatePaymentIntent(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (CreatePaymentIntent)")
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
	rideIDParam := c.Params("ride_id") // Assuming route is /rides/:ride_id/create-payment-intent
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for create intent: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	// 3. (Optional) Parse request body if needed in the future
	// var req models.CreatePaymentIntentRequest
	// if err := c.BodyParser(&req); err != nil { ... }

	log.Printf("Received create payment intent request from user %s for ride %s", userID, rideID)

	// 4. Call service to create payment intent
	response, err := h.paymentService.CreatePaymentIntent(c.Context(), rideID, userID)
	if err != nil {
		log.Printf("Error creating payment intent for user %s, ride %s: %v", userID, rideID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to create payment intent"
		// Handle specific errors from service
		errMsg := err.Error()
		// Use strings.Contains or errors.Is if error wrapping is used in service
		if errMsg == "user has not joined this ride or participation record not found" || errMsg == "cannot create payment for participation with status" {
			statusCode = http.StatusConflict // 409 Conflict
			errorMessage = errMsg
		} else if errMsg == "database error fetching participation record" {
			// Keep internal error
		} else if errMsg == "failed to create payment intent with Stripe" {
			// Keep internal error
		} else if errMsg == "failed to save transaction record" {
			// Keep internal error
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status":  "error",
			"message": errorMessage,
			// "details": err.Error(), // Optional
		})
	}

	// 5. Return successful response with client secret
	log.Printf("Payment intent created successfully for user %s, ride %s. Payment ID: %s", userID, rideID, response.PaymentID) // Use PaymentID
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Payment intent created successfully",
		"data":    response, // Contains client_secret, payment_id, amount, currency
	})
}

// CreateSetupIntent handles POST /api/v1/payments/setup-intent
// Requires authentication.
func (h *PaymentHandler) CreateSetupIntent(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (CreateSetupIntent)")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Missing user identification."})
		}
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context: %s", userIDStr)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": "Unauthorized: Invalid user identification format."})
		}
		userID = parsedID
	}

	log.Printf("Received create setup intent request from user %s", userID)

	// 2. Call service to create setup intent
	response, err := h.paymentService.CreateSetupIntent(c.Context(), userID)
	if err != nil {
		log.Printf("Error creating setup intent for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to create setup intent"
		// Use errors.Is for specific error checking if the service returns wrapped errors, e.g.:
		// if errors.Is(err, services.ErrUserNotFound) { ... }
		if err.Error() == "user not found" { // Simple string comparison used here
			statusCode = http.StatusNotFound
			errorMessage = err.Error()
		}
		// Add more specific error handling if needed
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	// 3. Return successful response with client secret and customer ID
	log.Printf("Setup intent created successfully for user %s. Customer ID: %s", userID, response.CustomerID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Setup intent created successfully",
		"data":    response, // Contains client_secret, customer_id
	})
}

// JoinRideAutomatically handles POST /api/v1/rides/:ride_id/join-automatic
// Requires authentication and saved payment method.
func (h *PaymentHandler) JoinRideAutomatically(c *fiber.Ctx) error {
	// 1. Get authenticated user ID from context
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		userIDStr, okStr := c.Locals("userID").(string)
		if !okStr {
			log.Println("Error: User ID not found in context (JoinRideAutomatically)")
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
	rideIDParam := c.Params("ride_id")
	rideID, err := uuid.Parse(rideIDParam)
	if err != nil {
		log.Printf("Invalid ride ID format in URL parameter for automatic join: %s", rideIDParam)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Invalid ride ID format"})
	}

	log.Printf("Received automatic join request from user %s for ride %s", userID, rideID)

	// 3. Call service to handle automatic join and payment
	err = h.paymentService.JoinRideAutomatically(c.Context(), rideID, userID)
	if err != nil {
		log.Printf("Error during automatic join for user %s, ride %s: %v", userID, rideID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to join ride automatically"

		// Handle specific errors from the service
		errMsg := err.Error()
		// TODO: Use errors.Is or specific error types if defined in service
		switch errMsg {
		case "user not found":
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		case "ride is full", "already joined", "cannot join your own ride": // Add other validation errors from service
			statusCode = http.StatusConflict // 409 Conflict for business logic errors
			errorMessage = errMsg
		case "user has no saved payment method setup":
			statusCode = http.StatusPaymentRequired // 402 Payment Required might be suitable
			errorMessage = errMsg
		case "payment failed": // Catch generic payment failure from service
			statusCode = http.StatusConflict // Or 400 Bad Request? Conflict seems okay.
			errorMessage = "Payment using saved method failed. Please update your payment details."
		case "critical error: payment processed but failed to update records":
			// This is a serious internal error, return 500 but log it critically
			errorMessage = "An internal error occurred while finalizing your participation."
		default:
			// Keep 500 for other unexpected errors
		}

		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	// 4. Return successful response
	log.Printf("Automatic join successful for user %s, ride %s", userID, rideID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Successfully joined ride and payment processed.",
	})
}

// HandleStripeWebhook is the conceptual handler for POST /api/v1/stripe-webhook
// The actual route registration in main.go needs to adapt this to a standard http.HandlerFunc.
func (h *PaymentHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Stripe webhook event via adapted handler")

	if r.Method != http.MethodPost {
		log.Printf("Webhook Error: Invalid method %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call the service method, passing the standard http.Request
	err := h.paymentService.HandleStripeWebhook(r)
	if err != nil {
		// Log the error from the service
		log.Printf("Error handling Stripe webhook: %v", err)
		// Return appropriate status code to Stripe based on the error
		// 400 for signature verification errors or bad requests
		// 500 for internal processing errors
		// Check error type if possible, otherwise default to 500 or 400
		// For simplicity, return 400 for now, as Stripe often retries on 5xx.
		http.Error(w, "Webhook processing failed", http.StatusBadRequest)
		return
	}

	// Return 200 OK to acknowledge receipt of the event
	log.Println("Webhook processed successfully (acknowledged)")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Webhook received successfully") // Optional body
}

// SetupPaymentRoutes registers the payment-related routes.
// Note the special handling needed for the webhook route.
func SetupPaymentRoutes(api fiber.Router, paymentService *services.PaymentService, authMiddleware fiber.Handler) {
	handler := NewPaymentHandler(paymentService)

	// Group for payment related routes under /payments
	paymentGroup := api.Group("/payments")

	// Route for creating setup intent (protected)
	paymentGroup.Post("/setup-intent", authMiddleware, handler.CreateSetupIntent)

	// Route for creating payment intent (protected) - Keep under /rides for context? Or move to /payments?
	// POST /api/v1/rides/:ride_id/create-payment-intent
	api.Post("/rides/:ride_id/create-payment-intent", authMiddleware, handler.CreatePaymentIntent) // For manual payment flow if needed later?
	api.Post("/rides/:ride_id/join-automatic", authMiddleware, handler.JoinRideAutomatically)      // New route for automatic payment

	log.Println("Payment routes (/payments/setup-intent, /rides/:ride_id/create-payment-intent, /rides/:ride_id/join-automatic) setup complete.")
	log.Println("Webhook route (/stripe-webhook) requires special registration in main.go using adaptor.HTTPHandler.")

}
