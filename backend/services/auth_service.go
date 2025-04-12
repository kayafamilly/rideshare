package services

import (
	"context" // For database operations context
	"errors"  // For creating standard errors
	"fmt"     // For string formatting
	"log"     // For logging
	"time"    // For time operations (JWT expiry)

	"github.com/go-playground/validator/v10" // For request validation
	"github.com/golang-jwt/jwt/v5"           // For JWT generation and validation
	"github.com/google/uuid"                 // For UUIDs
	"github.com/jackc/pgx/v5"                // For pgx specific errors (like no rows)
	"golang.org/x/crypto/bcrypt"             // For password hashing

	"rideshare/backend/config"   // Local config package
	"rideshare/backend/database" // Local database package
	"rideshare/backend/models"   // Local models package
)

// AuthService handles authentication logic.
type AuthService struct {
	cfg       *config.Config
	validator *validator.Validate
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		cfg:       cfg,
		validator: validator.New(), // Initialize validator
	}
}

// SignUp handles user registration.
func (s *AuthService) SignUp(ctx context.Context, req models.SignUpRequest) (*models.User, error) {
	// 1. Validate request data
	if err := s.validator.Struct(req); err != nil {
		log.Printf("Validation error during signup for email %s: %v", req.Email, err)
		return nil, fmtErrorf("invalid signup data: %w", err) // Return validation error
	}

	// 2. Check if email or WhatsApp number already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE (email = $1 OR whatsapp = $2) AND deleted_at IS NULL)` // Also check not deleted
	err := database.DB.QueryRow(ctx, checkQuery, req.Email, req.WhatsApp).Scan(&exists)
	if err != nil {
		log.Printf("Error checking user existence for email %s: %v", req.Email, err)
		return nil, fmtErrorf("database error checking user existence: %w", err)
	}
	if exists {
		log.Printf("Signup attempt failed: Email '%s' or WhatsApp '%s' already exists.", req.Email, req.WhatsApp)
		return nil, errors.New("email or WhatsApp number already registered") // User-friendly error
	}

	// 3. Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password for email %s: %v", req.Email, err)
		return nil, fmtErrorf("failed to hash password: %w", err)
	}

	// 4. Parse birth date
	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		log.Printf("Error parsing birth date '%s' for email %s: %v", req.BirthDate, req.Email, err)
		return nil, fmtErrorf("invalid birth date format (use YYYY-MM-DD): %w", err)
	}

	// 5. Create the user in the database
	newUser := &models.User{
		ID:           uuid.New(), // Generate new UUID
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    &req.FirstName, // Use pointers for optional fields
		LastName:     &req.LastName,
		BirthDate:    &birthDate,
		Nationality:  &req.Nationality,
		WhatsApp:     req.WhatsApp,
		// CreatedAt and UpdatedAt will be set by default in DB
		// DeletedAt is NULL by default
	}

	insertQuery := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	err = database.DB.QueryRow(ctx, insertQuery,
		newUser.ID, newUser.Email, newUser.PasswordHash, newUser.FirstName, newUser.LastName, newUser.BirthDate, newUser.Nationality, newUser.WhatsApp,
	).Scan(&newUser.CreatedAt, &newUser.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting new user for email %s: %v", req.Email, err)
		return nil, fmtErrorf("failed to create user in database: %w", err)
	}

	log.Printf("User created successfully: %s (ID: %s)", newUser.Email, newUser.ID)
	// Don't return password hash in the response model
	newUser.PasswordHash = ""
	return newUser, nil
}

// Login handles user login.
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	// 1. Validate request data
	if err := s.validator.Struct(req); err != nil {
		log.Printf("Validation error during login for email %s: %v", req.Email, err)
		return nil, fmtErrorf("invalid login data: %w", err)
	}

	// 2. Find the user by email (ensure not deleted)
	var user models.User
	query := `
		SELECT id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp, created_at, updated_at, stripe_customer_id
		FROM users WHERE email = $1 AND deleted_at IS NULL
	` // Added deleted_at check and stripe_customer_id
	// Use pointer for stripe_customer_id to handle NULL
	err := database.DB.QueryRow(ctx, query, req.Email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.BirthDate, &user.Nationality, &user.WhatsApp, &user.CreatedAt, &user.UpdatedAt, &user.StripeCustomerID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Login attempt failed: User not found or deleted for email %s", req.Email) // Updated log
			return nil, errors.New("invalid email or password")                                   // Generic error for security
		}
		log.Printf("Error fetching user during login for email %s: %v", req.Email, err)
		return nil, fmtErrorf("database error fetching user: %w", err)
	}

	// 3. Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		// Password doesn't match
		log.Printf("Login attempt failed: Invalid password for email %s", req.Email)
		return nil, errors.New("invalid email or password") // Generic error
	}

	// 4. Generate JWT token
	token, err := s.generateJWT(user.ID)
	if err != nil {
		log.Printf("Error generating JWT for user %s: %v", user.ID, err)
		return nil, fmtErrorf("failed to generate authentication token: %w", err)
	}

	log.Printf("User logged in successfully: %s (ID: %s)", user.Email, user.ID)

	// Prepare response (don't include password hash)
	// Determine if user has a payment method based on StripeCustomerID
	user.HasPaymentMethod = user.StripeCustomerID != nil && *user.StripeCustomerID != ""
	// Exclude sensitive fields from response
	user.PasswordHash = ""
	user.StripeCustomerID = nil // Don't send stripe customer id to frontend
	loginResponse := &models.LoginResponse{
		Token: token,
		User:  user,
	}

	return loginResponse, nil
}

// generateJWT creates a new JWT token for a given user ID.
func (s *AuthService) generateJWT(userID uuid.UUID) (string, error) {
	// Set custom claims
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Token expires after 72 hours
		"iat":     time.Now().Unix(),                     // Issued at time
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and return it as a string.
	// The secret key should be loaded from config and be strong!
	signedToken, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmtErrorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// Helper function to wrap errors (optional, can make error handling cleaner)
func fmtErrorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// --- New V2 Functions ---

// UpdateProfile handles updating user profile information.
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req models.UpdateProfileRequest) (*models.User, error) {
	// 1. Validate the request data (optional fields with specific formats)
	if err := s.validator.Struct(req); err != nil {
		log.Printf("Validation error during profile update for user %s: %v", userID, err)
		return nil, fmtErrorf("invalid profile data: %w", err)
	}

	// 2. Build the UPDATE query dynamically based on provided fields
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argID := 1 // Start arg index at 1

	// Add fields to update if they are provided (not nil)
	if req.FirstName != nil {
		query += fmt.Sprintf(", first_name = $%d", argID)
		args = append(args, *req.FirstName)
		argID++
	}
	if req.LastName != nil {
		query += fmt.Sprintf(", last_name = $%d", argID)
		args = append(args, *req.LastName)
		argID++
	}
	if req.BirthDate != nil {
		// Parse the date string first
		birthDate, err := time.Parse("2006-01-02", *req.BirthDate)
		if err != nil {
			log.Printf("Error parsing birth date '%s' during update for user %s: %v", *req.BirthDate, userID, err)
			return nil, fmtErrorf("invalid birth date format (use YYYY-MM-DD): %w", err)
		}
		query += fmt.Sprintf(", birth_date = $%d", argID)
		args = append(args, birthDate)
		argID++
	}
	if req.Nationality != nil {
		query += fmt.Sprintf(", nationality = $%d", argID)
		args = append(args, *req.Nationality)
		argID++
	}
	if req.WhatsApp != nil {
		// Check for WhatsApp uniqueness before adding to query (excluding the current user)
		var exists bool
		// Ensure we only check against other active users
		checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE whatsapp = $1 AND id != $2 AND deleted_at IS NULL)`
		err := database.DB.QueryRow(ctx, checkQuery, *req.WhatsApp, userID).Scan(&exists)
		if err != nil {
			log.Printf("Error checking WhatsApp uniqueness during update for user %s: %v", userID, err)
			return nil, fmtErrorf("database error checking whatsapp uniqueness: %w", err)
		}
		if exists {
			log.Printf("Profile update failed for user %s: WhatsApp number '%s' already registered by another user.", userID, *req.WhatsApp)
			return nil, errors.New("whatsapp number already registered")
		}
		query += fmt.Sprintf(", whatsapp = $%d", argID)
		args = append(args, *req.WhatsApp)
		argID++
	}

	// Check if any fields were actually provided for update
	if len(args) == 0 {
		log.Printf("No fields provided for profile update for user %s", userID)
		// Return current user data without performing an update? Or return an error?
		// Let's return an error indicating nothing was updated.
		return nil, errors.New("no update data provided")
	}

	// Add WHERE clause and RETURNING clause to get updated user data
	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", argID) // Ensure user is not deleted
	args = append(args, userID)
	query += ` RETURNING id, email, first_name, last_name, birth_date, nationality, whatsapp, created_at, updated_at`

	log.Printf("Executing profile update for user %s with query: %s", userID, query)

	// 3. Execute the update query
	var updatedUser models.User
	err := database.DB.QueryRow(ctx, query, args...).Scan(
		&updatedUser.ID, &updatedUser.Email, &updatedUser.FirstName, &updatedUser.LastName,
		&updatedUser.BirthDate, &updatedUser.Nationality, &updatedUser.WhatsApp,
		&updatedUser.CreatedAt, &updatedUser.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This could happen if the user ID doesn't exist or is already deleted
			log.Printf("Profile update failed: User %s not found or already deleted.", userID)
			return nil, errors.New("user not found or deleted")
		}
		log.Printf("Error updating profile for user %s: %v", userID, err)
		// Handle potential unique constraint violation on whatsapp if check above failed due to race condition? Unlikely but possible.
		return nil, fmtErrorf("failed to update profile in database: %w", err)
	}

	log.Printf("Profile updated successfully for user %s", userID)
	return &updatedUser, nil
}

// DeleteAccount performs a soft delete on the user account.
func (s *AuthService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	log.Printf("Attempting soft delete for user %s", userID)

	query := `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	tag, err := database.DB.Exec(ctx, query, userID)

	if err != nil {
		log.Printf("Error soft deleting user %s: %v", userID, err)
		return fmtErrorf("database error deleting account: %w", err)
	}

	if tag.RowsAffected() == 0 {
		log.Printf("Soft delete failed: User %s not found or already deleted.", userID)
		return errors.New("user not found or already deleted")
	}

	log.Printf("User %s soft deleted successfully.", userID)
	// TODO: Add logic to handle related data if necessary (e.g., cancel active rides/participations?)
	// For V2, just deleting the user might be sufficient as per requirements.
	return nil
}

// UpdateLocation updates the user's last known geographical location.
func (s *AuthService) UpdateLocation(ctx context.Context, userID uuid.UUID, latitude float64, longitude float64) error {
	log.Printf("Attempting to update location for user %s to Lat: %f, Lon: %f", userID, latitude, longitude)

	// Validate coordinates roughly (basic checks, more complex validation could be added)
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		log.Printf("Invalid coordinates provided for user %s: Lat=%f, Lon=%f", userID, latitude, longitude)
		return errors.New("invalid latitude or longitude provided")
	}

	// Use ST_MakePoint(longitude, latitude) for PostGIS POINT type
	// SRID 4326 corresponds to WGS 84
	query := `
		UPDATE users
		SET last_known_location = ST_SetSRID(ST_MakePoint($1, $2), 4326),
		    updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`
	tag, err := database.DB.Exec(ctx, query, longitude, latitude, userID) // Note: Longitude first for ST_MakePoint

	if err != nil {
		log.Printf("Error updating location for user %s: %v", userID, err)
		return fmtErrorf("database error updating location: %w", err)
	}

	if tag.RowsAffected() == 0 {
		log.Printf("Update location failed: User %s not found or already deleted.", userID)
		return errors.New("user not found or deleted")
	}

	log.Printf("Location updated successfully for user %s", userID)
	return nil
}

// RegisterPushToken saves or updates the Expo Push Token for a given user.
func (s *AuthService) RegisterPushToken(ctx context.Context, userID uuid.UUID, pushToken string) error {
	log.Printf("Attempting to register push token for user %s", userID)

	// Basic validation for the token (Expo tokens usually start with ExponentPushToken[...])
	if len(pushToken) < 10 { // Arbitrary basic check
		log.Printf("Invalid push token format provided for user %s: %s", userID, pushToken)
		return errors.New("invalid push token format")
	}

	query := `
		UPDATE users
		SET expo_push_token = $1,
		    updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	tag, err := database.DB.Exec(ctx, query, pushToken, userID)

	if err != nil {
		log.Printf("Error registering push token for user %s: %v", userID, err)
		return fmtErrorf("database error registering push token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		log.Printf("Register push token failed: User %s not found or already deleted.", userID)
		return errors.New("user not found or deleted")
	}

	log.Printf("Push token registered successfully for user %s", userID)
	return nil
}
