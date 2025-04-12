package database

import (
	"context" // For managing request context and cancellation signals
	"fmt"     // For string formatting
	"log"     // For logging messages
	"strings" // Import strings package for manipulation
	"time"    // For time-related operations (e.g., connection timeout)

	"rideshare/backend/config" // Import the local config package

	"github.com/jackc/pgx/v5"         // Base pgx package
	"github.com/jackc/pgx/v5/pgconn"  // For pgconn.PgTx
	"github.com/jackc/pgx/v5/pgxpool" // PostgreSQL driver and connection pool
)

// DBPool defines the interface for database operations we need.
// This allows mocking for tests. It includes methods from pgxpool.Pool.
type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
	Begin(ctx context.Context) (pgx.Tx, error) // Add Begin method for transactions
	// Add BeginTx if specific transaction options are needed
}

// DB holds the database connection pool interface.
var DB DBPool

// ConnectDB initializes the database connection pool using configuration.
func ConnectDB(cfg *config.Config) error {
	// Construct the database connection string using Supabase conventions
	// Example: postgresql://postgres:[YOUR-PASSWORD]@db.[YOUR-PROJECT-REF].supabase.co:5432/postgres
	// Extract project ref from the URL
	// Extract project reference using string manipulation
	if !strings.HasPrefix(cfg.SupabaseURL, "https://") || !strings.HasSuffix(cfg.SupabaseURL, ".supabase.co") {
		log.Printf("Error: Invalid SUPABASE_URL format: %s. Expected format: https://<ref>.supabase.co", cfg.SupabaseURL)
		return fmt.Errorf("invalid SUPABASE_URL format: %s", cfg.SupabaseURL)
	}
	projectRef := strings.TrimPrefix(cfg.SupabaseURL, "https://")
	projectRef = strings.TrimSuffix(projectRef, ".supabase.co")

	if projectRef == "" {
		log.Printf("Error: Could not extract project reference from SUPABASE_URL: %s", cfg.SupabaseURL)
		return fmt.Errorf("could not extract project reference from SUPABASE_URL")
	}
	log.Printf("Extracted Supabase project reference: %s", projectRef) // Log extracted ref

	// Construct the connection string
	// Note: Ensure the user is 'postgres' and the dbname is 'postgres' for standard Supabase setup
	connString := fmt.Sprintf("postgresql://postgres:%s@db.%s.supabase.co:5432/postgres",
		cfg.SupabaseDBPassword,
		projectRef,
	)

	log.Println("Attempting to connect to database...")

	// Configure the connection pool
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		log.Printf("Error parsing database connection string: %v\n", err)
		return fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Set connection pool settings (optional but recommended)
	config.MaxConns = 10                      // Maximum number of connections in the pool
	config.MinConns = 2                       // Minimum number of connections to keep open
	config.MaxConnLifetime = time.Hour        // Maximum lifetime of a connection
	config.MaxConnIdleTime = time.Minute * 30 // Maximum idle time for a connection
	config.HealthCheckPeriod = time.Minute    // How often to check connection health

	// Establish the connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Printf("Error connecting to the database: %v\n", err)
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Assign the pool to the global DB variable
	DB = pool

	// Test the connection
	err = DB.Ping(context.Background())
	if err != nil {
		log.Printf("Error pinging database: %v\n", err)
		DB.Close() // Close the pool if ping fails
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Println("Database connection pool established successfully!")
	return nil
}

// CloseDB closes the database connection pool.
// Should be called on application shutdown.
func CloseDB() {
	if DB != nil {
		log.Println("Closing database connection pool...")
		// Type assertion might be needed if CloseDB is called outside of tests
		// where DB could be the mock interface. However, in normal operation,
		// DB will be *pgxpool.Pool which has Close(). The interface ensures
		// the method exists.
		DB.Close()
		log.Println("Database connection pool closed.")
	}
}

// InitDB loads config and connects to the database.
// Exits the application if connection fails.
func InitDB() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := ConnectDB(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Optional: Add a defer statement in main() to call CloseDB() on exit
	// Example in main.go: defer database.CloseDB()
}
