package services

import (
	"context"
	"errors"
	"regexp" // For matching SQL queries in mock
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3" // Mocking library
	"golang.org/x/crypto/bcrypt"

	"rideshare/backend/config"
	"rideshare/backend/database" // To set the global DB variable with the mock
	"rideshare/backend/models"
)

// Helper function to create a mock database connection and auth service for tests
func setupAuthTest(t *testing.T) (*AuthService, pgxmock.PgxPoolIface) {
	// Create mock pool
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Failed to create mock pool: %v", err)
	}
	// Set the global DB variable to the mock pool for the service to use
	database.DB = mock

	// Create a dummy config for the service
	// Important: Use a consistent JWT secret for testing token generation/validation if needed
	testCfg := &config.Config{
		JWTSecret: "test-secret-key", // Use a fixed secret for tests
		// Add other config fields if the service uses them directly
	}
	authService := NewAuthService(testCfg)

	return authService, mock
}

// Test successful user signup
func TestAuthService_SignUp_Success(t *testing.T) {
	authService, mock := setupAuthTest(t)
	defer mock.Close() // Ensure mock connection is closed

	// Input data for signup
	req := models.SignUpRequest{
		Email:       "test@example.com",
		Password:    "password123",
		FirstName:   "Test",
		LastName:    "User",
		BirthDate:   "1990-01-01",
		Nationality: "Testland",
		WhatsApp:    "+1234567890",
	}
	parsedBirthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		t.Fatalf("Test setup failed: could not parse birth date: %v", err)
	}
	// Note: Hashing password here is just for potential comparison if needed,
	// the mock uses AnyArg() for the hash during insertion check.
	// hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	// --- Mock Expectations ---
	// 1. Expect check for existing user (email/whatsapp) - return false (not exists)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR whatsapp = $2)`)).
		WithArgs(req.Email, req.WhatsApp).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	// 2. Expect insertion of the new user - return timestamps
	// Use relaxed args matching for password hash and UUID as they are generated dynamically
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`)).
		WithArgs(pgxmock.AnyArg(), req.Email, pgxmock.AnyArg(), &req.FirstName, &req.LastName, &parsedBirthDate, &req.Nationality, req.WhatsApp).
		WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), time.Now()))

	// --- Execute Service Method ---
	user, err := authService.SignUp(context.Background(), req)

	// --- Assertions ---
	if err != nil {
		t.Errorf("Expected no error during signup, but got: %v", err)
	}
	if user == nil {
		t.Fatal("Expected user object, but got nil")
	}
	if user.Email != req.Email {
		t.Errorf("Expected user email %s, but got %s", req.Email, user.Email)
	}
	if user.PasswordHash != "" { // Ensure password hash is cleared for response
		t.Error("Expected password hash to be empty in response, but it was not")
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

// Test signup failure when email already exists
func TestAuthService_SignUp_EmailExists(t *testing.T) {
	authService, mock := setupAuthTest(t)
	defer mock.Close()

	req := models.SignUpRequest{
		Email:       "existing@example.com",
		Password:    "password123",
		FirstName:   "Existing",
		LastName:    "User",
		BirthDate:   "1990-01-01",
		Nationality: "Oldland",
		WhatsApp:    "+9876543210",
	}

	// Expect check for existing user - return true (exists)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 OR whatsapp = $2)`)).
		WithArgs(req.Email, req.WhatsApp).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	// Execute
	_, err := authService.SignUp(context.Background(), req)

	// Assertions
	if err == nil {
		t.Fatal("Expected an error due to existing email/whatsapp, but got nil")
	}
	expectedErrMsg := "email or WhatsApp number already registered"
	if !errors.Is(err, errors.New(expectedErrMsg)) && err.Error() != expectedErrMsg { // Check error message text
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrMsg, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

// Test successful user login
func TestAuthService_Login_Success(t *testing.T) {
	authService, mock := setupAuthTest(t)
	defer mock.Close()

	req := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	userID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	now := time.Now()
	testFirstName := "Test"
	testLastName := "User"
	testNationality := "Testland"
	testWhatsapp := "+1234567890"

	// --- Mock Expectations ---
	// 1. Expect query to find user by email - return user data
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp, created_at, updated_at
		FROM users WHERE email = $1
	`)).
		WithArgs(req.Email).
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "password_hash", "first_name", "last_name", "birth_date", "nationality", "whatsapp", "created_at", "updated_at"}).
			AddRow(userID, req.Email, string(hashedPassword), &testFirstName, &testLastName, &now, &testNationality, testWhatsapp, now, now)) // Use pointers for nullable fields

	// --- Execute Service Method ---
	loginResponse, err := authService.Login(context.Background(), req)

	// --- Assertions ---
	if err != nil {
		t.Errorf("Expected no error during login, but got: %v", err)
	}
	if loginResponse == nil {
		t.Fatal("Expected login response object, but got nil")
	}
	if loginResponse.User.Email != req.Email {
		t.Errorf("Expected user email %s, but got %s", req.Email, loginResponse.User.Email)
	}
	if loginResponse.Token == "" {
		t.Error("Expected a JWT token, but got an empty string")
	}

	// Optional: Validate JWT token structure/claims if needed
	token, _, err := new(jwt.Parser).ParseUnverified(loginResponse.Token, jwt.MapClaims{})
	if err != nil {
		t.Errorf("Failed to parse generated JWT token: %v", err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if claims["user_id"] != userID.String() {
			t.Errorf("Expected user_id %s in token claims, but got %s", userID.String(), claims["user_id"])
		}
		// Check expiry is roughly correct (within a small window)
		if expFloat, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(expFloat), 0)
			expectedExp := time.Now().Add(time.Hour * 72)
			if expTime.Before(expectedExp.Add(-time.Minute*5)) || expTime.After(expectedExp.Add(time.Minute*5)) {
				t.Errorf("Token expiry time %v is not within the expected range around %v", expTime, expectedExp)
			}
		} else {
			t.Errorf("Could not parse token expiry claim: %v", claims["exp"])
		}

	} else {
		t.Error("Failed to read token claims")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

// Test login failure with incorrect password
func TestAuthService_Login_IncorrectPassword(t *testing.T) {
	authService, mock := setupAuthTest(t)
	defer mock.Close()

	req := models.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}
	userID := uuid.New()
	// Store a DIFFERENT hash than what 'wrongpassword' would generate
	correctHashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword123"), bcrypt.DefaultCost)
	now := time.Now()
	testFirstName := "Test" // Add dummy data for all scanned columns
	testLastName := "User"
	testNationality := "Testland"
	testWhatsapp := "+1234567890"

	// Expect query to find user by email - return user data with the correct hash
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp, created_at, updated_at
		FROM users WHERE email = $1
	`)).
		WithArgs(req.Email).
		WillReturnRows(pgxmock.NewRows([]string{"id", "email", "password_hash", "first_name", "last_name", "birth_date", "nationality", "whatsapp", "created_at", "updated_at"}).
			AddRow(userID, req.Email, string(correctHashedPassword), &testFirstName, &testLastName, &now, &testNationality, testWhatsapp, now, now)) // Return the correct hash

	// Execute
	_, err := authService.Login(context.Background(), req)

	// Assertions
	if err == nil {
		t.Fatal("Expected an error due to incorrect password, but got nil")
	}
	expectedErrMsg := "invalid email or password" // Generic error message
	if !errors.Is(err, errors.New(expectedErrMsg)) && err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrMsg, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

// Test login failure when user is not found
func TestAuthService_Login_UserNotFound(t *testing.T) {
	authService, mock := setupAuthTest(t)
	defer mock.Close()

	req := models.LoginRequest{
		Email:    "notfound@example.com",
		Password: "password123",
	}

	// Expect query to find user by email - return ErrNoRows
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, password_hash, first_name, last_name, birth_date, nationality, whatsapp, created_at, updated_at
		FROM users WHERE email = $1
	`)).
		WithArgs(req.Email).
		WillReturnError(pgx.ErrNoRows) // Simulate user not found

	// Execute
	_, err := authService.Login(context.Background(), req)

	// Assertions
	if err == nil {
		t.Fatal("Expected an error due to user not found, but got nil")
	}
	expectedErrMsg := "invalid email or password" // Generic error message
	if !errors.Is(err, errors.New(expectedErrMsg)) && err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrMsg, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unfulfilled expectations: %s", err)
	}
}

// Add more tests:
// - SignUp with invalid data (validation errors) -> Requires passing mock validator or testing validation separately
// - SignUp with database error during insert
// - Login with database error during select
// - Test JWT generation edge cases (if any)
