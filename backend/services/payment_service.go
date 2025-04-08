package services

import (
	"context"
	"encoding/json" // For handling webhook JSON payload
	"errors"
	"fmt"
	"io" // For reading webhook request body
	"log"
	"net/http" // For webhook request object

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"         // For pgx errors
	"github.com/jackc/pgx/v5/pgxpool" // Import pgxpool for transaction interface check
	"github.com/stripe/stripe-go/v72" // Use specific version
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/webhook"

	"rideshare/backend/config"
	"rideshare/backend/database"
	"rideshare/backend/models"
)

const (
	// Fixed amount for joining a ride (2 EUR in cents)
	fixedPaymentAmount int64  = 200
	paymentCurrency    string = "eur"
)

// PaymentService handles payment logic using Stripe.
type PaymentService struct {
	cfg *config.Config
	db  database.DBPool
	// rideService *RideService // Potentially needed to update ride status?
}

// NewPaymentService creates a new PaymentService instance.
func NewPaymentService(cfg *config.Config, db database.DBPool) *PaymentService {
	// Stripe client is initialized globally in main.go
	return &PaymentService{
		cfg: cfg,
		db:  db,
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent and a corresponding transaction record.
func (s *PaymentService) CreatePaymentIntent(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*models.CreatePaymentIntentResponse, error) {
	log.Printf("Attempting to create PaymentIntent for user %s joining ride %s", userID, rideID)

	// 1. Verify the user's participation status (should be 'pending_payment')
	//    and get the participant ID.
	var participantID uuid.UUID
	var participantStatus string // Read status as string from DB
	// Use a transaction? Maybe not strictly necessary here, but could ensure consistency.
	// For now, separate queries.
	query := `SELECT id, status FROM participants WHERE user_id = $1 AND ride_id = $2`
	err := s.db.QueryRow(ctx, query, userID, rideID).Scan(&participantID, &participantStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("PaymentIntent creation failed: User %s has not joined ride %s", userID, rideID)
			// This implies the user didn't successfully complete the JoinRide step first.
			return nil, errors.New("user has not joined this ride or participation record not found")
		}
		log.Printf("Error fetching participant record for user %s, ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("database error fetching participation record: %w", err)
	}

	// Check if status allows payment intent creation (e.g., only if pending)
	// Check if status allows payment intent creation (must be 'pending_payment')
	if participantStatus != string(models.ParticipantStatusPendingPayment) {
		log.Printf("PaymentIntent creation failed: Participation status for user %s, ride %s is '%s', expected '%s'",
			userID, rideID, participantStatus, string(models.ParticipantStatusPendingPayment))
		return nil, fmt.Errorf("cannot create payment for participation with status: %s", participantStatus)
	}

	// 2. Create a transaction record in our database (status 'pending')
	// Note: The table is now 'payments', but the model struct might still be 'Transaction'
	// Let's assume we rename the struct later or create a new 'Payment' struct.
	// For now, using the existing Transaction struct name but mapping to 'payments' table.
	// We also need to use string for status.
	payment := &models.Payment{ // Use the new Payment struct
		ID:                    uuid.New(),
		UserID:                userID,
		RideID:                rideID,
		ParticipantID:         &participantID,              // Link to the participation record
		StripePaymentIntentID: "",                          // Will be filled after creating Stripe PI
		Status:                models.PaymentStatusPending, // Use the new constant
		Amount:                fixedPaymentAmount,
		Currency:              paymentCurrency,
	}
	insertTxQuery := `
		INSERT INTO payments (id, user_id, ride_id, participant_id, stripe_payment_intent_id, status, amount, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	// We insert with an empty PI ID first, then update it. Alternatively, create PI first.
	// Let's create PI first to ensure we have the ID.

	// 3. Create PaymentIntent with Stripe
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(fixedPaymentAmount),
		Currency: stripe.String(paymentCurrency),
		// Add payment method types if needed, e.g., card
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		// CaptureMethod: stripe.String(string(stripe.PaymentIntentCaptureMethodManual)), // If using manual capture
		// Add other params like Description, ReceiptEmail if needed
	}
	// Add metadata using the dedicated method
	params.AddMetadata("payment_id", payment.ID.String()) // Use payment_id for clarity
	params.AddMetadata("user_id", userID.String())
	params.AddMetadata("ride_id", rideID.String())
	params.AddMetadata("participant_id", participantID.String())

	pi, err := paymentintent.New(params)
	if err != nil {
		log.Printf("Error creating Stripe PaymentIntent for user %s, ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("failed to create payment intent with Stripe: %w", err)
	}

	log.Printf("Stripe PaymentIntent created: %s for user %s, ride %s", pi.ID, userID, rideID)

	// 4. Now insert the payment record with the Stripe PI ID
	payment.StripePaymentIntentID = pi.ID
	err = s.db.QueryRow(ctx, insertTxQuery,
		payment.ID, payment.UserID, payment.RideID, payment.ParticipantID,
		payment.StripePaymentIntentID, payment.Status, payment.Amount, payment.Currency,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting payment record for user %s, ride %s, PI %s: %v", userID, rideID, pi.ID, err)
		// Attempt to cancel the Stripe PaymentIntent if DB insert fails? Requires careful handling.
		// For now, log the inconsistency.
		// stripe.PaymentIntentCancelParams{ CancellationReason: stripe.String("internal_error") }
		// paymentintent.Cancel(pi.ID, &cancelParams)
		return nil, fmt.Errorf("failed to save transaction record: %w", err)
	}

	log.Printf("Payment record created: %s for PI %s", payment.ID, pi.ID)

	// 5. Return response to frontend
	response := &models.CreatePaymentIntentResponse{
		ClientSecret: pi.ClientSecret,
		PaymentID:    payment.ID, // Use PaymentID
		Amount:       payment.Amount,
		Currency:     payment.Currency,
	}
	return response, nil
}

// HandleStripeWebhook processes incoming webhook events from Stripe.
func (s *PaymentService) HandleStripeWebhook(request *http.Request) error {
	payload, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("Webhook Error: Failed reading request body: %v", err)
		return fmt.Errorf("error reading request body: %w", err)
	}
	defer request.Body.Close()

	// Verify the webhook signature
	event, err := webhook.ConstructEvent(payload, request.Header.Get("Stripe-Signature"), s.cfg.StripeWebhookSecret)
	if err != nil {
		log.Printf("Webhook Error: Signature verification failed: %v", err)
		return fmt.Errorf("webhook signature verification failed: %w", err) // Return error to indicate failure (Stripe might retry)
	}

	log.Printf("Webhook Received: Event Type %s, ID %s", event.Type, event.ID)

	// Handle the event based on its type
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Webhook Error: Failed unmarshaling payment_intent.succeeded data: %v", err)
			return fmt.Errorf("error parsing webhook JSON for succeeded PI: %w", err)
		}
		log.Printf("Webhook Handling: PaymentIntent Succeeded: %s", paymentIntent.ID)
		// Update transaction and participant status in DB
		return s.handlePaymentIntentSucceeded(context.Background(), &paymentIntent) // Use background context for webhook

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Webhook Error: Failed unmarshaling payment_intent.payment_failed data: %v", err)
			return fmt.Errorf("error parsing webhook JSON for failed PI: %w", err)
		}
		log.Printf("Webhook Handling: PaymentIntent Failed: %s, Reason: %s", paymentIntent.ID, paymentIntent.LastPaymentError)
		// Update transaction status to 'failed' in DB
		return s.handlePaymentIntentFailed(context.Background(), &paymentIntent)

	// TODO: Handle other event types if necessary (e.g., refunds)
	// case "charge.refunded":
	// ...

	default:
		log.Printf("Webhook Info: Unhandled event type: %s", event.Type)
	}

	return nil // Return nil for unhandled events to acknowledge receipt
}

// handlePaymentIntentSucceeded updates the database after a successful payment.
func (s *PaymentService) handlePaymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) error {
	// Check if the database pool supports transactions. pgxpool.Pool does.
	pool, ok := s.db.(*pgxpool.Pool)
	if !ok {
		log.Printf("Webhook Error: DB pool does not support transactions for PI succeeded %s", pi.ID)
		// Fallback or error? For atomicity, error is safer.
		return errors.New("database does not support transactions required for payment confirmation")
	}

	// Use a transaction for consistency
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Webhook Error: Failed to begin transaction for PI succeeded %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction begin failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Update Payment status to 'succeeded'
	updatePaymentQuery := `UPDATE payments SET status = $1, updated_at = NOW() WHERE stripe_payment_intent_id = $2 AND status = $3`
	tag, err := tx.Exec(ctx, updatePaymentQuery, string(models.PaymentStatusSucceeded), pi.ID, string(models.PaymentStatusPending)) // Use new constants
	if err != nil {
		log.Printf("Webhook Error: Failed updating payment status for PI %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		log.Printf("Webhook Warning: No pending payment found or already updated for PI %s", pi.ID)
		// This might happen if the webhook is delivered multiple times. It's okay to ignore.
		// Or it could indicate an issue if the transaction was never created properly.
		// Consider querying the transaction status first.
	} else {
		log.Printf("Webhook DB Update: Payment status updated to succeeded for PI %s", pi.ID)
	}

	// 2. Update Participant status to 'confirmed'
	// Get participant ID from transaction metadata or by joining tables? Metadata is easier if reliable.
	// For robustness, let's find the participant via the payment table link.
	var participantID uuid.UUID
	findParticipantQuery := `SELECT participant_id FROM payments WHERE stripe_payment_intent_id = $1` // Already correct table name
	// Use QueryRow from the transaction (tx)
	err = tx.QueryRow(ctx, findParticipantQuery, pi.ID).Scan(&participantID)
	if err != nil {
		// Handle case where participant_id might be NULL or transaction not found (though checked above)
		log.Printf("Webhook Error: Could not find participant ID linked to PI %s: %v", pi.ID, err)
		// If participant_id is crucial, return error. If optional, maybe just log.
		return fmt.Errorf("could not find participant for PI %s: %w", pi.ID, err)
	}

	updateParticipantQuery := `UPDATE participants SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3`
	// Use Exec from the transaction (tx)
	tag, err = tx.Exec(ctx, updateParticipantQuery, string(models.ParticipantStatusActive), participantID, string(models.ParticipantStatusPendingPayment)) // Use string status 'active'
	if err != nil {
		log.Printf("Webhook Error: Failed updating participant status for ID %s (PI %s): %v", participantID, pi.ID, err)
		return fmt.Errorf("db participant update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		log.Printf("Webhook Warning: No pending participant found or already updated for ID %s (PI %s)", participantID, pi.ID)
		// Similar to transaction, could be duplicate webhook or prior issue.
	} else {
		log.Printf("Webhook DB Update: Participant status updated to active for ID %s (PI %s)", participantID, pi.ID)
	}

	// TODO: Add logic here for Step 5: Unlock WhatsApp numbers
	// This might involve fetching user details for both participants (creator and joiner)
	// and potentially sending a notification or updating another status.
	log.Printf("Webhook TODO: Implement WhatsApp number unlocking logic for PI %s", pi.ID)

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Webhook Error: Failed to commit transaction for PI succeeded %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction commit failed: %w", err)
	}

	log.Printf("Webhook Handling Complete: Successfully processed payment_intent.succeeded for %s", pi.ID)
	return nil
}

// handlePaymentIntentFailed updates the database after a failed payment.
func (s *PaymentService) handlePaymentIntentFailed(ctx context.Context, pi *stripe.PaymentIntent) error {
	// Update Transaction status to 'failed'
	// No need to update participant status usually, they remain 'pending_payment' or could be moved to 'payment_failed' if needed.
	updatePaymentQuery := `UPDATE payments SET status = $1, updated_at = NOW() WHERE stripe_payment_intent_id = $2 AND status = $3`
	tag, err := s.db.Exec(ctx, updatePaymentQuery, string(models.PaymentStatusFailed), pi.ID, string(models.PaymentStatusPending)) // Use new constants
	if err != nil {
		log.Printf("Webhook Error: Failed updating payment status to failed for PI %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		log.Printf("Webhook Warning: No pending payment found or already updated for failed PI %s", pi.ID)
	} else {
		log.Printf("Webhook DB Update: Payment status updated to failed for PI %s", pi.ID)
	}

	log.Printf("Webhook Handling Complete: Successfully processed payment_intent.payment_failed for %s", pi.ID)
	return nil
}
