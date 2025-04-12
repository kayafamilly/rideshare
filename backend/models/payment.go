package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the possible statuses of a payment.
// Corresponds to the 'status' column in the 'payments' table.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"   // Initial status before Stripe confirmation
	PaymentStatusSucceeded PaymentStatus = "succeeded" // Payment confirmed by Stripe webhook
	PaymentStatusFailed    PaymentStatus = "failed"    // Payment failed according to Stripe webhook
	// TransactionStatusRefunded is removed as it's not in the simplified V2 status list
)

// Payment represents the structure for the 'payments' table (renamed from 'transactions').
type Payment struct {
	ID                    uuid.UUID     `json:"id" db:"id"`
	UserID                uuid.UUID     `json:"user_id" db:"user_id"`                                   // User who paid
	RideID                uuid.UUID     `json:"ride_id" db:"ride_id"`                                   // Associated ride
	ParticipantID         *uuid.UUID    `json:"participant_id,omitempty" db:"participant_id"`           // Associated participation record (nullable)
	StripePaymentIntentID string        `json:"stripe_payment_intent_id" db:"stripe_payment_intent_id"` // Stripe Payment Intent ID
	Status                PaymentStatus `json:"status" db:"status"`                                     // Use new type
	Amount                int64         `json:"amount" db:"amount"`                                     // Amount in smallest currency unit (e.g., cents)
	Currency              string        `json:"currency" db:"currency"`                                 // 3-letter ISO currency code
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at" db:"updated_at"`
}

// --- DTOs ---

// CreatePaymentIntentRequest defines data needed from the frontend to create a payment intent.
// Currently, only the ride ID is needed as the amount is fixed (2 EUR).
// The user ID comes from the authenticated context.
type CreatePaymentIntentRequest struct {
	// We might add items here later if needed (e.g., specific payment method types)
}

// CreatePaymentIntentResponse defines the data sent back to the frontend.
type CreatePaymentIntentResponse struct {
	ClientSecret string    `json:"client_secret"` // The client secret of the PaymentIntent
	PaymentID    uuid.UUID `json:"payment_id"`    // Our internal payment ID (renamed field)
	Amount       int64     `json:"amount"`        // Amount to be paid
	Currency     string    `json:"currency"`      // Currency
	// Add publishable key if not already available on frontend? Usually set during init.
	// StripePublicKey string `json:"stripe_public_key"`
}

// CreateSetupIntentResponse defines the data sent back for setting up a payment method.
type CreateSetupIntentResponse struct {
	ClientSecret string `json:"client_secret"` // The client secret of the SetupIntent
	CustomerID   string `json:"customer_id"`   // The Stripe Customer ID
}
