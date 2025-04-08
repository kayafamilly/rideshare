package models

import (
	"time" // Package for time operations

	"github.com/google/uuid" // Package for UUID generation
)

// User represents the structure for the 'users' table in the database.
// Corresponds to the SQL definition in 001_create_users_table.sql
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`                             // Primary key, auto-generated UUID
	Email        string     `json:"email" db:"email"`                       // User's email, unique and required
	PasswordHash string     `json:"-" db:"password_hash"`                   // Hashed password (excluded from JSON responses)
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`   // User's first name (optional, pointer for NULL)
	LastName     *string    `json:"last_name,omitempty" db:"last_name"`     // User's last name (optional, pointer for NULL)
	BirthDate    *time.Time `json:"birth_date,omitempty" db:"birth_date"`   // User's date of birth (optional, pointer for NULL)
	Nationality  *string    `json:"nationality,omitempty" db:"nationality"` // User's nationality (optional, pointer for NULL)
	WhatsApp     string     `json:"whatsapp" db:"whatsapp"`                 // User's WhatsApp number, unique and required
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`             // Timestamp of user creation
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`             // Timestamp of last user update
	DeletedAt    *time.Time `json:"-" db:"deleted_at"`                      // Timestamp for soft delete (excluded from JSON)
}

// SignUpRequest defines the structure for user registration requests.
// Contains fields required for creating a new user account.
type SignUpRequest struct {
	Email       string `json:"email" validate:"required,email"`                    // User's email address
	Password    string `json:"password" validate:"required,min=8"`                 // User's chosen password (min 8 chars)
	FirstName   string `json:"first_name" validate:"required"`                     // User's first name
	LastName    string `json:"last_name" validate:"required"`                      // User's last name
	BirthDate   string `json:"birth_date" validate:"required,datetime=2006-01-02"` // User's birth date (YYYY-MM-DD format)
	Nationality string `json:"nationality" validate:"required"`                    // User's nationality
	WhatsApp    string `json:"whatsapp" validate:"required,e164"`                  // User's WhatsApp number (E.164 format validation)
}

// LoginRequest defines the structure for user login requests.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"` // User's email address
	Password string `json:"password" validate:"required"`    // User's password
}

// LoginResponse defines the structure for successful login responses.
// Typically includes a session token and basic user info.
type LoginResponse struct {
	Token string `json:"token"` // Authentication token (e.g., JWT)
	User  User   `json:"user"`  // Basic user information (excluding sensitive data like password hash)
}

// UpdateProfileRequest defines the structure for updating user profile information.
// All fields are optional, only provided fields will be updated.
type UpdateProfileRequest struct {
	FirstName   *string `json:"first_name,omitempty"`                                          // Optional: New first name
	LastName    *string `json:"last_name,omitempty"`                                           // Optional: New last name
	BirthDate   *string `json:"birth_date,omitempty" validate:"omitempty,datetime=2006-01-02"` // Optional: New birth date (YYYY-MM-DD)
	Nationality *string `json:"nationality,omitempty"`                                         // Optional: New nationality
	WhatsApp    *string `json:"whatsapp,omitempty" validate:"omitempty,e164"`                  // Optional: New WhatsApp number (E.164)
	// Email/Password changes might require separate flows for security (e.g., verification)
}

// Note: Added SignUpRequest, LoginRequest, LoginResponse, UpdateProfileRequest.
// Added basic validation tags using 'validate' struct tags.
// Used pointers for optional fields in User struct and UpdateProfileRequest.
// Excluded PasswordHash and DeletedAt from JSON responses using `json:"-"`.
