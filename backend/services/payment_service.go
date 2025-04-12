package services

import (
	"context"
	"database/sql"  // Import database/sql for sql.NullString
	"encoding/json" // For handling webhook JSON payload
	"errors"
	"fmt"
	"io" // For reading webhook request body
	"log"
	"net/http" // For webhook request object

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"         // For pgx errors
	"github.com/jackc/pgx/v5/pgconn"  // Import pgconn for PgError type
	"github.com/jackc/pgx/v5/pgxpool" // Import pgxpool for transaction interface check
	"github.com/stripe/stripe-go/v72" // Use specific version
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

// StripeService defines the interface for interacting with the Stripe API.
// This allows for mocking in tests.
type StripeService interface {
	CreateCustomer(ctx context.Context, params *stripe.CustomerParams) (*stripe.Customer, error)
	CreateSetupIntent(ctx context.Context, params *stripe.SetupIntentParams) (*stripe.SetupIntent, error)
	CreatePaymentIntent(ctx context.Context, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error)
	CreateAndConfirmPaymentIntent(ctx context.Context, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error)
	ConstructWebhookEvent(payload []byte, signatureHeader string, secret string) (stripe.Event, error)
}

// PaymentService handles payment logic using Stripe.
type PaymentService struct {
	cfg          *config.Config
	db           database.DBPool
	rideService  *RideService  // Inject RideService
	stripeClient StripeService // Inject Stripe client interface
}

// NewPaymentService creates a new PaymentService instance.
func NewPaymentService(cfg *config.Config, db database.DBPool, rideService *RideService, stripeClient StripeService) *PaymentService {
	return &PaymentService{
		cfg:          cfg,
		db:           db,
		rideService:  rideService,  // Store injected RideService
		stripeClient: stripeClient, // Store injected Stripe client
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent and a corresponding transaction record.
func (s *PaymentService) CreatePaymentIntent(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*models.CreatePaymentIntentResponse, error) {
	log.Printf("Attempting to create PaymentIntent for user %s joining ride %s", userID, rideID)

	// 1. Verify the user's participation status (should be 'pending_payment')
	//    and get the participant ID.
	var participantID uuid.UUID
	var participantStatus string // Read status as string from DB
	query := `SELECT id, status FROM participants WHERE user_id = $1 AND ride_id = $2`
	err := s.db.QueryRow(ctx, query, userID, rideID).Scan(&participantID, &participantStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("PaymentIntent creation failed: User %s has not joined ride %s", userID, rideID)
			return nil, errors.New("user has not joined this ride or participation record not found")
		}
		log.Printf("Error fetching participant record for user %s, ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("database error fetching participation record: %w", err)
	}

	// Check if status allows payment intent creation (must be 'pending_payment')
	if participantStatus != string(models.ParticipantStatusPendingPayment) {
		log.Printf("PaymentIntent creation failed: Participation status for user %s, ride %s is '%s', expected '%s'",
			userID, rideID, participantStatus, string(models.ParticipantStatusPendingPayment))
		return nil, fmt.Errorf("cannot create payment for participation with status: %s", participantStatus)
	}

	// 2. Create a transaction record in our database (status 'pending')
	payment := &models.Payment{
		ID:                    uuid.New(),
		UserID:                userID,
		RideID:                rideID,
		ParticipantID:         &participantID,
		StripePaymentIntentID: "", // Will be filled after creating Stripe PI
		Status:                models.PaymentStatusPending,
		Amount:                fixedPaymentAmount,
		Currency:              paymentCurrency,
	}

	// 3. Create PaymentIntent with Stripe
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(fixedPaymentAmount),
		Currency:           stripe.String(paymentCurrency),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}
	params.AddMetadata("payment_id", payment.ID.String())
	params.AddMetadata("user_id", userID.String())
	params.AddMetadata("ride_id", rideID.String())
	params.AddMetadata("participant_id", participantID.String())

	pi, err := s.stripeClient.CreatePaymentIntent(ctx, params)
	if err != nil {
		log.Printf("Error creating Stripe PaymentIntent for user %s, ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("failed to create payment intent with Stripe: %w", err)
	}
	log.Printf("Stripe PaymentIntent created: %s for user %s, ride %s", pi.ID, userID, rideID)

	// 4. Now insert the payment record with the Stripe PI ID
	payment.StripePaymentIntentID = pi.ID
	insertTxQuery := `
		INSERT INTO payments (id, user_id, ride_id, participant_id, stripe_payment_intent_id, status, amount, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	err = s.db.QueryRow(ctx, insertTxQuery,
		payment.ID, payment.UserID, payment.RideID, payment.ParticipantID,
		payment.StripePaymentIntentID, payment.Status, payment.Amount, payment.Currency,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting payment record for user %s, ride %s, PI %s: %v", userID, rideID, pi.ID, err)
		// Consider attempting to cancel the Stripe PaymentIntent here
		return nil, fmt.Errorf("failed to save transaction record: %w", err)
	}
	log.Printf("Payment record created: %s for PI %s", payment.ID, pi.ID)

	// 5. Return response to frontend
	response := &models.CreatePaymentIntentResponse{
		ClientSecret: pi.ClientSecret,
		PaymentID:    payment.ID,
		Amount:       payment.Amount,
		Currency:     payment.Currency,
	}
	return response, nil
}

// CreateSetupIntent finds or creates a Stripe Customer for the user and creates a SetupIntent.
func (s *PaymentService) CreateSetupIntent(ctx context.Context, userID uuid.UUID) (*models.CreateSetupIntentResponse, error) {
	log.Printf("Attempting to create SetupIntent for user %s", userID)

	// 1. Find or Create Stripe Customer ID
	var stripeCustomerID sql.NullString
	var userEmail string
	var userFirstName string
	var userLastName string

	queryUser := `SELECT email, first_name, last_name, stripe_customer_id FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.QueryRow(ctx, queryUser, userID).Scan(&userEmail, &userFirstName, &userLastName, &stripeCustomerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("SetupIntent creation failed: User %s not found", userID)
			return nil, errors.New("user not found")
		}
		log.Printf("Error fetching user %s for SetupIntent: %v", userID, err)
		return nil, fmt.Errorf("database error fetching user: %w", err)
	}

	if !stripeCustomerID.Valid || stripeCustomerID.String == "" {
		log.Printf("Stripe Customer ID not found for user %s. Creating new Stripe Customer.", userID)
		customerParams := &stripe.CustomerParams{
			Email: stripe.String(userEmail),
			Name:  stripe.String(fmt.Sprintf("%s %s", userFirstName, userLastName)),
		}
		customerParams.AddMetadata("app_user_id", userID.String())

		newCustomer, err := s.stripeClient.CreateCustomer(ctx, customerParams)
		if err != nil {
			log.Printf("Error creating Stripe Customer for user %s: %v", userID, err)
			return nil, fmt.Errorf("failed to create stripe customer: %w", err)
		}
		log.Printf("Stripe Customer created: %s for user %s", newCustomer.ID, userID)
		stripeCustomerID.String = newCustomer.ID
		stripeCustomerID.Valid = true

		updateUserQuery := `UPDATE users SET stripe_customer_id = $1, updated_at = NOW() WHERE id = $2`
		_, err = s.db.Exec(ctx, updateUserQuery, stripeCustomerID.String, userID)
		if err != nil {
			log.Printf("CRITICAL Error: Failed to update user %s with Stripe Customer ID %s: %v", userID, stripeCustomerID.String, err)
			// Proceed even if update fails, SetupIntent might still work
		}
	}

	// 2. Create SetupIntent for the customer
	setupParams := &stripe.SetupIntentParams{
		Customer:           stripe.String(stripeCustomerID.String),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		Usage:              stripe.String(string(stripe.SetupIntentUsageOffSession)),
	}
	setupParams.AddMetadata("app_user_id", userID.String())

	si, err := s.stripeClient.CreateSetupIntent(ctx, setupParams)
	if err != nil {
		log.Printf("Error creating Stripe SetupIntent for user %s (Customer %s): %v", userID, stripeCustomerID.String, err)
		return nil, fmt.Errorf("failed to create setup intent: %w", err)
	}
	log.Printf("Stripe SetupIntent created: %s for user %s", si.ID, userID)

	// 3. Return response
	response := &models.CreateSetupIntentResponse{
		ClientSecret: si.ClientSecret,
		CustomerID:   stripeCustomerID.String,
	}
	return response, nil
}

// HandleStripeWebhook processes incoming webhook events from Stripe.
func (s *PaymentService) HandleStripeWebhook(request *http.Request) error {
	log.Println("--- HandleStripeWebhook invoked ---") // Log entry

	payload, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("!!! Webhook Error STEP 1 (Read Body): %v", err)
		// Return error to indicate failure to Stripe
		return fmt.Errorf("error reading request body: %w", err)
	}
	defer request.Body.Close()
	log.Println("--- Webhook STEP 2: Body read successfully ---")

	signature := request.Header.Get("Stripe-Signature")
	log.Printf("--- Webhook STEP 3a: Verifying signature: %s ---", signature)
	event, err := webhook.ConstructEvent(payload, signature, s.cfg.StripeWebhookSecret)
	if err != nil {
		log.Printf("!!! Webhook Error STEP 3b (ConstructEvent/Verify Signature): %v", err)
		// Return error to indicate failure to Stripe
		return fmt.Errorf("webhook signature verification failed: %w", err)
	}
	log.Printf("--- Webhook STEP 4: Event constructed successfully (Type: %s, ID: %s) ---", event.Type, event.ID)

	// Handle the event based on its type
	switch event.Type {
	case "payment_intent.succeeded":
		log.Printf("--- Webhook STEP 5a: Handling event type %s ---", event.Type)
		var paymentIntent stripe.PaymentIntent // Declare here
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("!!! Webhook Error STEP 5b (Unmarshal %s): %v", event.Type, err)
			return fmt.Errorf("error parsing webhook JSON for %s: %w", event.Type, err)
		}
		log.Printf("Webhook Handling: PaymentIntent Succeeded: %s", paymentIntent.ID)
		return s.handlePaymentIntentSucceeded(context.Background(), &paymentIntent)

	case "payment_intent.payment_failed":
		log.Printf("--- Webhook STEP 5a: Handling event type %s ---", event.Type)
		var paymentIntent stripe.PaymentIntent // Declare here
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("!!! Webhook Error STEP 5b (Unmarshal %s): %v", event.Type, err)
			return fmt.Errorf("error parsing webhook JSON for %s: %w", event.Type, err)
		}
		log.Printf("Webhook Handling: PaymentIntent Failed: %s, Reason: %s", paymentIntent.ID, paymentIntent.LastPaymentError)
		return s.handlePaymentIntentFailed(context.Background(), &paymentIntent)

	case "setup_intent.succeeded":
		log.Printf("--- Webhook STEP 5a: Handling event type %s ---", event.Type)
		var setupIntent stripe.SetupIntent // Declare here
		err := json.Unmarshal(event.Data.Raw, &setupIntent)
		if err != nil {
			log.Printf("!!! Webhook Error STEP 5b (Unmarshal %s): %v", event.Type, err)
			return fmt.Errorf("error parsing webhook JSON for %s: %w", event.Type, err)
		}
		log.Printf("Webhook Handling: SetupIntent Succeeded: %s", setupIntent.ID)
		return s.handleSetupIntentSucceeded(context.Background(), &setupIntent)

	default:
		log.Printf("Webhook Info: Unhandled event type: %s", event.Type)
	}

	return nil // Return nil for unhandled events to acknowledge receipt
}

// handleSetupIntentSucceeded processes the setup_intent.succeeded webhook event.
func (s *PaymentService) handleSetupIntentSucceeded(ctx context.Context, si *stripe.SetupIntent) error {
	customerID := ""
	if si.Customer != nil {
		customerID = si.Customer.ID
	}
	appUserID := si.Metadata["app_user_id"]

	paymentMethodID := ""
	if si.PaymentMethod != nil {
		paymentMethodID = si.PaymentMethod.ID
	}

	if paymentMethodID == "" {
		log.Printf("Webhook Warning: SetupIntent %s succeeded but PaymentMethod ID is missing.", si.ID)
		return nil
	}

	if appUserID != "" {
		updateUserQuery := `
			UPDATE users
			SET stripe_default_payment_method_id = $1,
			    has_payment_method = TRUE,
			    updated_at = NOW()
			WHERE id = $2`
		log.Printf("--- Attempting DB Update for user %s with PM %s ---", appUserID, paymentMethodID)
		tag, err := s.db.Exec(ctx, updateUserQuery, paymentMethodID, appUserID)
		if err != nil {
			log.Printf("Webhook CRITICAL Error: Failed to update user %s with default payment method %s after SI %s succeeded: %v",
				appUserID, paymentMethodID, si.ID, err)
			return fmt.Errorf("failed to update user default payment method: %w", err)
		}
		log.Printf("--- DB Update Exec Result: RowsAffected=%d, Error=%v ---", tag.RowsAffected(), err)
		log.Printf("Webhook DB Update: User %s default payment method updated to %s for SI %s", appUserID, paymentMethodID, si.ID)
	} else {
		log.Printf("Webhook Warning: Cannot update user's default payment method for SI %s because app_user_id metadata is missing.", si.ID)
	}

	log.Printf("Webhook Handling Complete: Successfully processed setup_intent.succeeded for SI %s, Customer %s, App User %s, PM %s",
		si.ID, customerID, appUserID, paymentMethodID)

	return nil
}

// handlePaymentIntentSucceeded updates the database after a successful payment.
func (s *PaymentService) handlePaymentIntentSucceeded(ctx context.Context, pi *stripe.PaymentIntent) error {
	pool, ok := s.db.(*pgxpool.Pool)
	if !ok {
		log.Printf("Webhook Error: DB pool does not support transactions for PI succeeded %s", pi.ID)
		return errors.New("database does not support transactions required for payment confirmation")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Webhook Error: Failed to begin transaction for PI succeeded %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction begin failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Update Payment status to 'succeeded'
	updatePaymentQuery := `UPDATE payments SET status = $1, updated_at = NOW() WHERE stripe_payment_intent_id = $2 AND status = $3`
	tag, err := tx.Exec(ctx, updatePaymentQuery, string(models.PaymentStatusSucceeded), pi.ID, string(models.PaymentStatusPending))
	if err != nil {
		log.Printf("Webhook Error: Failed updating payment status for PI %s: %v", pi.ID, err)
		return fmt.Errorf("db transaction update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		log.Printf("Webhook Warning: No pending payment found or already updated for PI %s", pi.ID)
	} else {
		log.Printf("Webhook DB Update: Payment status updated to succeeded for PI %s", pi.ID)
	}

	// 2. Update Participant status to 'active'
	var participantID uuid.UUID
	findParticipantQuery := `SELECT participant_id FROM payments WHERE stripe_payment_intent_id = $1`
	err = tx.QueryRow(ctx, findParticipantQuery, pi.ID).Scan(&participantID)
	if err != nil {
		log.Printf("Webhook Error: Could not find participant ID linked to PI %s: %v", pi.ID, err)
		return fmt.Errorf("could not find participant for PI %s: %w", pi.ID, err)
	}

	updateParticipantQuery := `UPDATE participants SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3`
	tag, err = tx.Exec(ctx, updateParticipantQuery, string(models.ParticipantStatusActive), participantID, string(models.ParticipantStatusPendingPayment))
	if err != nil {
		log.Printf("Webhook Error: Failed updating participant status for ID %s (PI %s): %v", participantID, pi.ID, err)
		return fmt.Errorf("db participant update failed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		log.Printf("Webhook Warning: No pending participant found or already updated for ID %s (PI %s)", participantID, pi.ID)
	} else {
		log.Printf("Webhook DB Update: Participant status updated to active for ID %s (PI %s)", participantID, pi.ID)
	}

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
	updatePaymentQuery := `UPDATE payments SET status = $1, updated_at = NOW() WHERE stripe_payment_intent_id = $2 AND status = $3`
	tag, err := s.db.Exec(ctx, updatePaymentQuery, string(models.PaymentStatusFailed), pi.ID, string(models.PaymentStatusPending))
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

// JoinRideAutomatically attempts to join a user to a ride and charge their saved payment method.
func (s *PaymentService) JoinRideAutomatically(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) error {
	log.Printf("Attempting automatic join for user %s on ride %s", userID, rideID)

	// --- Database Transaction ---
	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Printf("Automatic Join Error: Failed to begin transaction for user %s, ride %s: %v", userID, rideID, err)
		return fmt.Errorf("db transaction begin failed: %w", err)
	}
	defer tx.Rollback(ctx)

	// --- 1. Validation (using RideService within the transaction) ---
	_, err = s.rideService.ValidateRideForJoiningTx(ctx, tx, rideID, userID)
	if err != nil {
		return err // Validation failed (e.g., full, already joined, etc.)
	}

	// --- 2. Get Stripe Customer ID and Default Payment Method ---
	var stripeCustomerID sql.NullString
	var stripeDefaultPaymentMethodID sql.NullString
	queryUser := `SELECT stripe_customer_id, stripe_default_payment_method_id FROM users WHERE id = $1 AND deleted_at IS NULL`
	err = tx.QueryRow(ctx, queryUser, userID).Scan(&stripeCustomerID, &stripeDefaultPaymentMethodID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Automatic Join Error: User %s not found", userID)
			return errors.New("user not found")
		}
		log.Printf("Automatic Join Error: Failed fetching Stripe details for user %s: %v", userID, err)
		return fmt.Errorf("database error fetching user details: %w", err)
	}
	if !stripeCustomerID.Valid || stripeCustomerID.String == "" {
		log.Printf("Automatic Join Error: User %s has no Stripe customer ID", userID)
		return errors.New("user has no Stripe customer ID setup")
	}
	if !stripeDefaultPaymentMethodID.Valid || stripeDefaultPaymentMethodID.String == "" {
		log.Printf("Automatic Join Error: User %s has no default payment method ID set", userID)
		return errors.New("user has no saved default payment method")
	}
	customerID := stripeCustomerID.String
	paymentMethodID := stripeDefaultPaymentMethodID.String
	log.Printf("Automatic Join Info: Found Stripe Customer ID %s and PM ID %s for user %s", customerID, paymentMethodID, userID)

	// --- 3. Check for existing participation record (especially 'left' status) ---
	var existingParticipant models.Participant // Declare here
	checkParticipantQuery := `SELECT id, status FROM participants WHERE user_id = $1 AND ride_id = $2`
	err = tx.QueryRow(ctx, checkParticipantQuery, userID, rideID).Scan(&existingParticipant.ID, &existingParticipant.Status) // Scan into the declared variable

	var participantIDToUse uuid.UUID
	var needsPayment bool = true // Assume payment is needed unless rejoining

	if err == nil { // Record found
		switch existingParticipant.Status {
		case string(models.ParticipantStatusActive), string(models.ParticipantStatusPendingPayment):
			log.Printf("Automatic Join Error: User %s already has participation record with status '%s' for ride %s", userID, existingParticipant.Status, rideID)
			return fmt.Errorf("user already participating with status: %s", existingParticipant.Status)
		case string(models.ParticipantStatusLeft):
			log.Printf("Automatic Join Info: User %s previously left ride %s. Updating status to active.", userID, rideID)
			updateStatusQuery := `UPDATE participants SET status = $1, updated_at = NOW() WHERE id = $2`
			_, updateErr := tx.Exec(ctx, updateStatusQuery, string(models.ParticipantStatusActive), existingParticipant.ID)
			if updateErr != nil {
				log.Printf("Automatic Join Error: Failed updating status for rejoining participant %s on ride %s: %v", userID, rideID, updateErr)
				return fmt.Errorf("failed to update participation status for rejoin: %w", updateErr)
			}
			participantIDToUse = existingParticipant.ID
			needsPayment = false // User is rejoining, no new payment needed
		default:
			log.Printf("Automatic Join Error: User %s has an unexpected participation status '%s' for ride %s", userID, rideID, existingParticipant.Status)
			return fmt.Errorf("unexpected participation status: %s", existingParticipant.Status)
		}
	} else if errors.Is(err, pgx.ErrNoRows) {
		// No existing record, insert a new one
		log.Printf("Automatic Join Info: No existing participation found for user %s on ride %s. Inserting new record.", userID, rideID)
		participant := &models.Participant{
			ID:     uuid.New(),
			RideID: rideID,
			UserID: userID,
			Status: string(models.ParticipantStatusActive), // Set to Active directly as payment will be attempted now
		}
		insertParticipantQuery := `INSERT INTO participants (id, ride_id, user_id, status) VALUES ($1, $2, $3, $4)`
		_, insertErr := tx.Exec(ctx, insertParticipantQuery, participant.ID, participant.RideID, participant.UserID, participant.Status)
		if insertErr != nil {
			log.Printf("Automatic Join Error: Failed inserting participant record for user %s, ride %s: %v", userID, rideID, insertErr)
			var pgErr *pgconn.PgError
			if errors.As(insertErr, &pgErr) && pgErr.Code == "23505" { // unique_violation
				log.Printf("Automatic Join Error: Unique constraint violation despite check for user %s, ride %s.", userID, rideID)
				return errors.New("participation record conflict")
			}
			return fmt.Errorf("database error inserting participant: %w", insertErr)
		}
		participantIDToUse = participant.ID
		needsPayment = true // New participant, needs payment
	} else {
		// Actual database error during check
		log.Printf("Automatic Join Error: Failed checking existing participation for user %s, ride %s: %v", userID, rideID, err)
		return fmt.Errorf("database error checking participation: %w", err)
	}

	// --- 4. Create and Confirm PaymentIntent (Off-Session) ONLY IF NEEDED ---
	var pi *stripe.PaymentIntent // Declare pi outside the block

	if needsPayment {
		piParams := &stripe.PaymentIntentParams{
			Amount:                stripe.Int64(fixedPaymentAmount),
			Currency:              stripe.String(paymentCurrency),
			Customer:              stripe.String(customerID),
			PaymentMethod:         stripe.String(paymentMethodID),
			Confirm:               stripe.Bool(true),
			OffSession:            stripe.Bool(true),
			ErrorOnRequiresAction: stripe.Bool(true),
		}
		piParams.AddMetadata("app_user_id", userID.String())
		piParams.AddMetadata("ride_id", rideID.String())
		piParams.AddMetadata("charge_type", "automatic_join_new")

		pi, err = s.stripeClient.CreateAndConfirmPaymentIntent(ctx, piParams)
		if err != nil {
			log.Printf("Automatic Join Error: Stripe PaymentIntent creation/confirmation failed for user %s, ride %s: %v", userID, rideID, err)
			// Rollback should happen automatically due to defer tx.Rollback(ctx)
			return fmt.Errorf("payment failed: %w", err)
		}

		if pi.Status != stripe.PaymentIntentStatusSucceeded {
			log.Printf("Automatic Join Error: PaymentIntent status is %s, expected succeeded for user %s, ride %s, PI %s", pi.Status, userID, rideID, pi.ID)
			// Rollback should happen automatically
			return fmt.Errorf("payment confirmation failed with status: %s", pi.Status)
		}
		log.Printf("Automatic Join Info: Stripe PaymentIntent %s succeeded for user %s, ride %s", pi.ID, userID, rideID)

		// --- 5. Insert payment record ONLY IF payment was made ---
		payment := &models.Payment{
			ID:                    uuid.New(),
			UserID:                userID,
			RideID:                rideID,
			ParticipantID:         &participantIDToUse,
			StripePaymentIntentID: pi.ID,
			Status:                models.PaymentStatusSucceeded,
			Amount:                fixedPaymentAmount,
			Currency:              paymentCurrency,
		}
		insertPaymentQuery := `INSERT INTO payments (id, user_id, ride_id, participant_id, stripe_payment_intent_id, status, amount, currency) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = tx.Exec(ctx, insertPaymentQuery, payment.ID, payment.UserID, payment.RideID, payment.ParticipantID, payment.StripePaymentIntentID, payment.Status, payment.Amount, payment.Currency)
		if err != nil {
			log.Printf("Automatic Join Error: Failed inserting payment record for user %s, ride %s, PI %s: %v", userID, rideID, pi.ID, err)
			// Rollback should happen automatically
			return fmt.Errorf("database error inserting payment: %w", err)
		}
		log.Printf("Automatic Join Info: Payment record inserted for user %s, ride %s, PI %s", userID, rideID, pi.ID)

	} else {
		log.Printf("Automatic Join Info: Skipping Stripe payment and payment record insertion for rejoining user %s, ride %s", userID, rideID)
	}

	// --- 6. Commit Transaction ---
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Automatic Join Error: Failed to commit transaction for user %s, ride %s: %v", userID, rideID, err)
		// Critical: Payment might have succeeded but DB update failed if needsPayment was true.
		// If needsPayment was false, commit failure is less critical but still an issue.
		return fmt.Errorf("critical error: failed to finalize participation records")
	}

	log.Printf("Automatic Join Success: User %s successfully joined/rejoined ride %s", userID, rideID)
	return nil // Success
}
