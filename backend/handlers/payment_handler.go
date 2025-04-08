package handlers

import (
	"fmt" // Import fmt
	"log"
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

	// Route for creating payment intent (protected)
	// POST /api/v1/rides/:ride_id/create-payment-intent
	api.Post("/rides/:ride_id/create-payment-intent", authMiddleware, handler.CreatePaymentIntent)

	// Route for Stripe Webhook (public)
	// This route registration will be different in main.go using adaptor.HTTPHandler
	// The handler logic is defined above in HandleStripeWebhook (as http.HandlerFunc).
	// We don't register it directly on the Fiber router here.

	log.Println("Payment routes (/rides/:ride_id/create-payment-intent) setup complete.")
	log.Println("Webhook route (/stripe-webhook) requires special registration in main.go using adaptor.HTTPHandler.")

}
