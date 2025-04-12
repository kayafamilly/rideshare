package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"rideshare/backend/models"
	"rideshare/backend/services"
)

// AuthHandler handles HTTP requests related to authentication and user management.
type AuthHandler struct {
	authService *services.AuthService
}

// RegisterPushTokenRequest defines the structure for the push token registration request.
type RegisterPushTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignUp handles the POST /api/v1/auth/signup request.
func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	var req models.SignUpRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing signup request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
			"details": err.Error(),
		})
	}
	log.Printf("Received signup request for email: %s", req.Email)

	user, err := h.authService.SignUp(c.Context(), req)
	if err != nil {
		log.Printf("Error during signup process for email %s: %v", req.Email, err)
		statusCode := fiber.StatusInternalServerError
		errorMessage := "Signup failed due to an internal error"
		errMsg := err.Error()
		if errMsg == "email or WhatsApp number already registered" {
			statusCode = fiber.StatusConflict
			errorMessage = errMsg
		} else if errMsg == "invalid birth date format (use YYYY-MM-DD)" {
			statusCode = fiber.StatusBadRequest
			errorMessage = errMsg
		} else {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = fiber.StatusBadRequest
				errorMessage = fmt.Sprintf("Invalid signup data: %v", validationErrors)
			}
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Signup successful for user: %s (ID: %s)", user.Email, user.ID)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "User registered successfully",
		"data":    user,
	})
}

// Login handles the POST /api/v1/auth/login request.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing login request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Invalid request body", "details": err.Error(),
		})
	}
	log.Printf("Received login request for email: %s", req.Email)

	loginResponse, err := h.authService.Login(c.Context(), req)
	if err != nil {
		log.Printf("Error during login process for email %s: %v", req.Email, err)
		statusCode := fiber.StatusInternalServerError
		errorMessage := "Login failed due to an internal error"
		errMsg := err.Error()
		if errMsg == "invalid email or password" {
			statusCode = fiber.StatusUnauthorized
			errorMessage = errMsg
		} else {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = fiber.StatusBadRequest
				errorMessage = fmt.Sprintf("Invalid login data: %v", validationErrors)
			}
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Login successful for user: %s (ID: %s)", loginResponse.User.Email, loginResponse.User.ID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success", "message": "Login successful", "data": loginResponse,
	})
}

// Helper to get userID from context, handling potential string or uuid.UUID types
func getUserIDFromContext(c *fiber.Ctx, handlerName string) (uuid.UUID, error) {
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		log.Printf("Error: User ID not found in context (%s)", handlerName)
		return uuid.Nil, errors.New("unauthorized: Missing user identification")
	}

	switch id := userIDLocal.(type) {
	case uuid.UUID:
		return id, nil
	case string:
		parsedID, err := uuid.Parse(id)
		if err != nil {
			log.Printf("Error: Invalid User ID format in context (%s): %s", handlerName, id)
			return uuid.Nil, errors.New("unauthorized: Invalid user identification format")
		}
		return parsedID, nil
	default:
		log.Printf("Error: Unexpected User ID type in context (%s): %T", handlerName, userIDLocal)
		return uuid.Nil, errors.New("unauthorized: Unexpected user identification type")
	}
}

// UpdateProfile handles PUT /api/v1/users/profile
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c, "UpdateProfile")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	var req models.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing update profile request body for user %s: %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Invalid request body", "details": err.Error(),
		})
	}
	log.Printf("Received update profile request from user %s: %+v", userID, req)

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
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		} else {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = http.StatusBadRequest
				errorMessage = fmt.Sprintf("Invalid profile data: %v", validationErrors)
			}
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Profile updated successfully for user %s", userID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "success", "message": "Profile updated successfully", "data": updatedUser,
	})
}

// DeleteAccount handles DELETE /api/v1/users/account
func (h *AuthHandler) DeleteAccount(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c, "DeleteAccount")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}
	log.Printf("Received delete account request from user %s", userID)

	err = h.authService.DeleteAccount(c.Context(), userID)
	if err != nil {
		log.Printf("Error deleting account for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to delete account"
		if err.Error() == "user not found or already deleted" {
			statusCode = http.StatusNotFound
			errorMessage = err.Error()
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Account deleted successfully for user %s", userID)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "success", "message": "Account deleted successfully",
	})
}

// UpdateLocation handles PUT /api/v1/users/location
func (h *AuthHandler) UpdateLocation(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c, "UpdateLocation")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	var req models.UpdateLocationRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing update location request body for user %s: %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Invalid request body", "details": err.Error(),
		})
	}
	log.Printf("Received update location request from user %s: Lat=%f, Lon=%f", userID, req.Latitude, req.Longitude)

	err = h.authService.UpdateLocation(c.Context(), userID, req.Latitude, req.Longitude)
	if err != nil {
		log.Printf("Error updating location for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to update location"
		errMsg := err.Error()
		if errMsg == "user not found or deleted" {
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		} else if errMsg == "invalid latitude or longitude provided" {
			statusCode = http.StatusBadRequest
			errorMessage = errMsg
		} else {
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				statusCode = http.StatusBadRequest
				errorMessage = fmt.Sprintf("Invalid location data: %v", validationErrors)
			}
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Location updated successfully for user %s", userID)
	return c.SendStatus(http.StatusNoContent)
}

// RegisterPushToken handles POST /api/v1/users/push-token
func (h *AuthHandler) RegisterPushToken(c *fiber.Ctx) error {
	userID, err := getUserIDFromContext(c, "RegisterPushToken")
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": err.Error()})
	}

	var req RegisterPushTokenRequest // Use the struct defined at package level
	if err := c.BodyParser(&req); err != nil {
		log.Printf("Error parsing register push token request body for user %s: %v", userID, err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"status": "error", "message": "Invalid request body", "details": err.Error(),
		})
	}

	// Basic validation on the token itself
	if req.Token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "Push token cannot be empty"})
	}
	log.Printf("Received register push token request from user %s", userID)

	err = h.authService.RegisterPushToken(c.Context(), userID, req.Token)
	if err != nil {
		log.Printf("Error registering push token for user %s: %v", userID, err)
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to register push token"
		errMsg := err.Error()
		if errMsg == "user not found or deleted" {
			statusCode = http.StatusNotFound
			errorMessage = errMsg
		} else if errMsg == "invalid push token format" {
			statusCode = http.StatusBadRequest
			errorMessage = errMsg
		}
		return c.Status(statusCode).JSON(fiber.Map{"status": "error", "message": errorMessage})
	}

	log.Printf("Push token registered successfully for user %s", userID)
	return c.SendStatus(http.StatusNoContent)
}

// SetupUserRoutes registers user profile and account management routes.
func SetupUserRoutes(api fiber.Router, authService *services.AuthService, authMiddleware fiber.Handler) {
	handler := NewAuthHandler(authService)
	userGroup := api.Group("/users")
	userGroup.Put("/profile", authMiddleware, handler.UpdateProfile)
	userGroup.Delete("/account", authMiddleware, handler.DeleteAccount)
	userGroup.Put("/location", authMiddleware, handler.UpdateLocation)
	userGroup.Post("/push-token", authMiddleware, handler.RegisterPushToken) // Register the new route
	log.Println("User routes (/users/profile, /users/account, /users/location, /users/push-token) setup complete.")
}

// SetupAuthRoutes registers the public authentication routes.
func SetupAuthRoutes(api fiber.Router, authService *services.AuthService) {
	handler := NewAuthHandler(authService)
	authGroup := api.Group("/auth")
	authGroup.Post("/signup", handler.SignUp)
	authGroup.Post("/login", handler.Login)
	log.Println("Authentication routes (/auth/signup, /auth/login) setup complete.")
}
