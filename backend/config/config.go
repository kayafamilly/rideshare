package config

import (
	"log" // Standard log package
	"os"  // Package to interact with the OS, including environment variables

	"github.com/joho/godotenv" // Package to load .env files
)

// Config holds all configuration for the application.
// Values are read from environment variables.
type Config struct {
	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string
	SupabaseDBPassword     string // Added Database password
	StripeSecretKey        string
	StripePublicKey        string
	StripeWebhookSecret    string
	ServerPort             string
	JWTSecret              string // Added for signing JWT tokens
}

// LoadConfig reads configuration from environment variables.
// It loads a .env file first if it exists.
func LoadConfig() (*Config, error) {
	// Attempt to load .env file. Ignore error if it doesn't exist.
	err := godotenv.Load() // Loads .env from the current directory
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Read environment variables or use defaults
	cfg := &Config{
		SupabaseURL:            getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:        getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseDBPassword:     getEnv("SUPABASE_DB_PASSWORD", ""), // Read DB password
		StripeSecretKey:        getEnv("STRIPE_SECRET_KEY", ""),
		StripePublicKey:        getEnv("STRIPE_PUBLIC_KEY", ""),
		StripeWebhookSecret:    getEnv("STRIPE_WEBHOOK_SECRET", ""),
		ServerPort:             getEnv("SERVER_PORT", "8080"),                // Default port 8080
		JWTSecret:              getEnv("JWT_SECRET", "your-very-secret-key"), // !! CHANGE THIS IN PRODUCTION !!
	}

	// Basic validation (ensure critical keys are present)
	if cfg.SupabaseURL == "" || cfg.SupabaseServiceRoleKey == "" || cfg.SupabaseDBPassword == "" || cfg.StripeSecretKey == "" || cfg.JWTSecret == "your-very-secret-key" {
		log.Println("Warning: One or more critical configuration keys (Supabase URL/Service Key/DB Password, Stripe Secret, JWT Secret) are missing or using default values.")
		// In a real app, you might return an error here or handle it more robustly.
		// For JWT_SECRET, it's crucial to set a strong, unique secret via environment variables.
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Environment variable %s not set, using fallback '%s'", key, fallback)
	return fallback
}
