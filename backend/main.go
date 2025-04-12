package main

import (
	"log" // Import standard log package

	"github.com/gofiber/adaptor/v2"                 // Fiber adaptor for net/http handlers
	"github.com/gofiber/fiber/v2"                   // Import Fiber framework
	"github.com/gofiber/fiber/v2/middleware/logger" // Fiber logger middleware

	"rideshare/backend/config"     // Local config package
	"rideshare/backend/database"   // Local database package
	"rideshare/backend/handlers"   // Local handlers package
	"rideshare/backend/middleware" // Local middleware package
	"rideshare/backend/services"   // Local services package

	"github.com/stripe/stripe-go/v72" // Stripe Go client (adjust version if needed)
	// webhook package is needed by payment_service, not directly here if using adaptor
)

// main is the entry point of the application.
func main() {
	// Load configuration first
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	database.InitDB()        // This also loads config, but we load it above for clarity and potential use
	defer database.CloseDB() // Ensure DB connection is closed when main function exits

	// Initialize Stripe client
	stripe.Key = cfg.StripeSecretKey
	log.Println("Stripe client initialized with configured secret key.")

	// Create a new Fiber app instance
	app := fiber.New()

	// Add logger middleware for http requests
	app.Use(logger.New())

	// Simple health check route at the root
	app.Get("/", func(c *fiber.Ctx) error {
		log.Println("Health check '/' accessed")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok", "message": "Welcome to RideShare Backend!"})
	})

	// Setup API v1 group
	apiV1 := app.Group("/api/v1")
	log.Println("API group /api/v1 setup")

	// --- Setup application services ---
	authService := services.NewAuthService(cfg)
	// Pass the database pool interface to NewRideService
	rideService := services.NewRideService(database.DB)
	stripeService := services.NewStripeServiceImpl()                                           // Create real Stripe service implementation
	paymentService := services.NewPaymentService(cfg, database.DB, rideService, stripeService) // Inject rideService and stripeService

	// --- Setup middleware ---
	authMiddleware := middleware.Protected(cfg) // Create auth middleware instance

	// --- Setup routes ---
	handlers.SetupAuthRoutes(apiV1, authService)
	handlers.SetupRideRoutes(apiV1, rideService, authMiddleware)
	handlers.SetupPaymentRoutes(apiV1, paymentService, authMiddleware) // This sets up payment routes EXCEPT webhook
	handlers.SetupUserRoutes(apiV1, authService, authMiddleware)       // Add user routes

	// --- Setup Stripe Webhook Route using net/http adaptor ---
	// Create a separate http handler instance for the webhook
	webhookHandler := handlers.NewPaymentHandler(paymentService) // Need instance for method reference
	// Adapt the http.HandlerFunc to Fiber's handler type
	// The path MUST match the one configured in your Stripe dashboard
	app.Post("/api/v1/stripe-webhook", adaptor.HTTPHandlerFunc(webhookHandler.HandleStripeWebhook)) // Use the adaptor
	log.Println("Stripe webhook route (/api/v1/stripe-webhook) registered using adaptor.")

	// Use port from configuration
	port := cfg.ServerPort
	log.Printf("Starting RideShare backend server on port %s", port)

	// Start the Fiber server
	// Listen on the specified port, log fatal error if server fails to start
	log.Fatal(app.Listen(":" + port))
}
