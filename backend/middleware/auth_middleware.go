package middleware

import (
	"errors" // Import errors package
	"log"
	"strings" // For string manipulation (Bearer token)

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid" // For parsing UUID from token

	"rideshare/backend/config" // To get JWT secret
)

// Protected is a middleware function to protect routes that require authentication.
// It verifies the JWT token from the Authorization header.
func Protected(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Println("Auth Middleware: Missing Authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Missing authorization token",
			})
		}

		// Check if the header format is "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Println("Auth Middleware: Invalid Authorization header format")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Invalid token format",
			})
		}

		tokenString := parts[1]

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				log.Printf("Auth Middleware: Unexpected signing method: %v", token.Header["alg"])
				return nil, jwt.ErrSignatureInvalid // Or a more specific error
			}
			// Return the secret key for validation
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil {
			log.Printf("Auth Middleware: Error parsing or validating token: %v", err)
			// Handle specific JWT errors (e.g., expired token)
			if errors.Is(err, jwt.ErrTokenExpired) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Unauthorized: Token has expired",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Unauthorized: Invalid token",
				// "details": err.Error(), // Avoid leaking detailed errors
			})
		}

		// Check if token is valid and extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Extract user ID from claims
			userIDStr, ok := claims["user_id"].(string)
			if !ok {
				log.Println("Auth Middleware: 'user_id' claim missing or not a string in token")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Unauthorized: Invalid token claims (missing user_id)",
				})
			}

			// Parse UUID
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				log.Printf("Auth Middleware: Failed to parse user_id claim '%s' as UUID: %v", userIDStr, err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"status":  "error",
					"message": "Unauthorized: Invalid token claims (invalid user_id format)",
				})
			}

			// Store user ID in locals for subsequent handlers
			c.Locals("userID", userID) // Store as uuid.UUID
			// c.Locals("userID_str", userIDStr) // Optionally store string version too if needed elsewhere
			log.Printf("Auth Middleware: User %s authenticated successfully.", userID)

			// Token is valid, proceed to the next handler
			return c.Next()
		}

		// Token is invalid for some other reason
		log.Println("Auth Middleware: Token deemed invalid.")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized: Invalid token",
		})
	}
}
