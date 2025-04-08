package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"         // For pgx errors
	"github.com/jackc/pgx/v5/pgxpool" // Import pgxpool for transaction interface check

	"rideshare/backend/database"
	"rideshare/backend/models"
)

// RideService handles business logic related to rides.
type RideService struct {
	validator *validator.Validate
	db        database.DBPool // Use the DBPool interface
	// Add other dependencies if needed, e.g., notification service
}

// NewRideService creates a new RideService instance.
func NewRideService(db database.DBPool) *RideService {
	return &RideService{
		validator: validator.New(),
		db:        db, // Store the db pool/mock
	}
}

// CreateRide handles the creation of a new ride.
func (s *RideService) CreateRide(ctx context.Context, req models.CreateRideRequest, userID uuid.UUID) (*models.Ride, error) {
	// 1. Validate request data
	if err := s.validator.Struct(req); err != nil {
		log.Printf("Validation error creating ride for user %s: %v", userID, err)
		return nil, fmt.Errorf("invalid ride data: %w", err)
	}

	// 2. Parse date and time strings
	departureDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		log.Printf("Error parsing departure date '%s' for user %s: %v", req.DepartureDate, userID, err)
		return nil, fmt.Errorf("invalid departure date format (use YYYY-MM-DD): %w", err)
	}
	// Note: We store time as string HH:MM, matching the DB TIME type. No parsing needed here for insertion.
	// Ensure the input format is validated (done via validator tag 'datetime=15:04').

	// Basic check: Departure date/time should be in the future (optional but good practice)
	// Combine date and time for comparison (assuming local timezone for simplicity)
	// More robust timezone handling might be needed.
	// Use current date if only time is provided? For now, require full date/time string.
	// Let's refine this logic: Combine parsed date with HH:MM time string.
	layout := "2006-01-02 15:04" // Layout for parsing combined date and time
	departureDateTimeStr := fmt.Sprintf("%s %s", req.DepartureDate, req.DepartureTime)
	departureDateTime, err := time.Parse(layout, departureDateTimeStr)
	// Consider server's local timezone. If times are meant to be UTC, adjust parsing/storage.
	// For simplicity, assume server time is the reference for "now".
	if err != nil {
		// This shouldn't happen if date and time formats are validated correctly, but handle defensively.
		log.Printf("Error combining departure date and time '%s %s' for user %s: %v", req.DepartureDate, req.DepartureTime, userID, err)
		return nil, fmt.Errorf("invalid departure date or time format: %w", err)
	}

	if departureDateTime.Before(time.Now()) {
		log.Printf("Validation error: Departure date/time %s is in the past for user %s", departureDateTime, userID)
		return nil, errors.New("departure date and time must be in the future")
	}

	// 3. Create the ride in the database
	newRide := &models.Ride{
		ID:            uuid.New(),
		UserID:        userID,
		StartLocation: req.StartLocation,
		EndLocation:   req.EndLocation,
		DepartureDate: departureDate,                   // Store the parsed date
		DepartureTime: req.DepartureTime,               // Store the HH:MM string
		TotalSeats:    req.TotalSeats,                  // Use TotalSeats from request
		Status:        string(models.RideStatusActive), // Default status is 'active' (string)
	}

	insertQuery := `
		INSERT INTO rides (id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	err = s.db.QueryRow(ctx, insertQuery, // Use s.db
		newRide.ID, newRide.UserID, newRide.StartLocation, newRide.EndLocation, newRide.DepartureDate, newRide.DepartureTime, newRide.TotalSeats, newRide.Status,
	).Scan(&newRide.CreatedAt, &newRide.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting new ride for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to create ride in database: %w", err)
	}

	log.Printf("Ride created successfully by user %s: Ride ID %s", userID, newRide.ID)
	return newRide, nil
}

// ListAvailableRides retrieves a list of rides that are currently 'open'.
// TODO: Add filtering (by date, location) and pagination.
func (s *RideService) ListAvailableRides(ctx context.Context) ([]models.Ride, error) {
	rides := []models.Ride{}
	// Query only for 'open' rides, order by departure date/time
	// Also filter out rides where the combined date and time is in the past
	query := `
		SELECT id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status, created_at, updated_at
		FROM rides
		WHERE status = $1 AND (departure_date > current_date OR (departure_date = current_date AND departure_time > current_time))
		ORDER BY departure_date ASC, departure_time ASC
	`
	// Note: Using current_date and current_time relies on the database server's time.

	rows, err := s.db.Query(ctx, query, string(models.RideStatusActive)) // Use s.db and new status
	if err != nil {
		log.Printf("Error querying available rides: %v", err)
		return nil, fmt.Errorf("database error fetching rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ride models.Ride
		// Scan each column into the Ride struct fields
		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
			&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats, // Use TotalSeats
			&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning ride row: %v", err)
			// Decide whether to return partial results or error out
			return nil, fmt.Errorf("error processing ride data: %w", err)
		}
		rides = append(rides, ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating ride rows: %v", err)
		return nil, fmt.Errorf("database iteration error: %w", err)
	}

	log.Printf("Fetched %d available rides", len(rides))
	return rides, nil
}

// GetRideDetails retrieves details for a specific ride by its ID.
// TODO: Optionally join with users table to get creator details.
func (s *RideService) GetRideDetails(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	var ride models.Ride
	query := `
		SELECT id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status, created_at, updated_at
		FROM rides
		WHERE id = $1
	`
	err := s.db.QueryRow(ctx, query, rideID).Scan( // Use s.db
		&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
		&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats, // Use TotalSeats
		&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Ride not found: ID %s", rideID)
			return nil, errors.New("ride not found")
		}
		log.Printf("Error fetching ride details for ID %s: %v", rideID, err)
		return nil, fmt.Errorf("database error fetching ride details: %w", err)
	}

	log.Printf("Fetched details for ride ID %s", rideID)
	return &ride, nil
}

// JoinRide allows a user to join an existing ride.
// This creates a 'participant' record with 'pending_payment' status.
func (s *RideService) JoinRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*models.Participant, error) {
	// Check if the database pool supports transactions. pgxpool.Pool does.
	pool, ok := s.db.(*pgxpool.Pool)
	if !ok {
		// If using a mock or a DB type that doesn't support transactions, handle it.
		// For tests with pgxmock, transactions need specific expectation setup.
		// For now, assume we need a real pool or a mock supporting transactions.
		log.Println("Warning: Database pool does not support transactions, proceeding without.")
		// Fallback to non-transactional logic or return error? For safety, return error.
		return nil, errors.New("database does not support transactions required for JoinRide")
		// OR: Implement non-transactional version (less safe for concurrency)
	}

	// --- Transaction Start ---
	tx, err := pool.Begin(ctx) // Use the asserted pool to begin transaction
	if err != nil {
		log.Printf("Error starting transaction for joining ride %s by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	// Defer rollback in case of errors, commit will override if successful.
	defer tx.Rollback(ctx)

	// 1. Get ride details and lock the row for update (to prevent race conditions on available_seats)
	var ride models.Ride
	lockQuery := `
		SELECT id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status
		FROM rides
		WHERE id = $1
		FOR UPDATE
	`
	// Use tx for operations within the transaction
	err = tx.QueryRow(ctx, lockQuery, rideID).Scan(
		&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
		&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats, &ride.Status, // Use TotalSeats
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("JoinRide failed: Ride not found: ID %s", rideID)
			return nil, errors.New("ride not found")
		}
		log.Printf("Error fetching/locking ride %s for join by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("database error fetching ride: %w", err)
	}

	// 2. Check ride status and availability
	// Check ride status (must be active)
	if ride.Status != string(models.RideStatusActive) {
		log.Printf("JoinRide failed: Ride %s is not active (status: %s)", rideID, ride.Status)
		return nil, errors.New("ride is not active for joining")
	}

	// Check if ride is full by comparing total seats with current active participants
	var activeParticipantsCount int
	countQuery := `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND status = $2`
	err = tx.QueryRow(ctx, countQuery, rideID, string(models.ParticipantStatusActive)).Scan(&activeParticipantsCount)
	if err != nil {
		log.Printf("Error counting active participants for ride %s: %v", rideID, err)
		return nil, fmt.Errorf("database error checking ride capacity: %w", err)
	}

	if activeParticipantsCount >= ride.TotalSeats {
		log.Printf("JoinRide failed: Ride %s is full (%d/%d seats taken)", rideID, activeParticipantsCount, ride.TotalSeats)
		// Optionally update status to 'archived' or similar if needed, but not 'full' status anymore
		return nil, errors.New("ride is already full")
	}
	if ride.UserID == userID {
		log.Printf("JoinRide failed: User %s cannot join their own ride %s", userID, rideID)
		return nil, errors.New("you cannot join your own ride")
	}

	// 3. Check if user has already joined this ride
	var existingParticipantID uuid.UUID
	checkParticipantQuery := `SELECT id FROM participants WHERE user_id = $1 AND ride_id = $2`
	err = tx.QueryRow(ctx, checkParticipantQuery, userID, rideID).Scan(&existingParticipantID)
	if err == nil { // If scan succeeds, a record exists
		log.Printf("JoinRide failed: User %s has already joined ride %s (Participant ID: %s)", userID, rideID, existingParticipantID)
		return nil, errors.New("you have already joined this ride")
	} else if !errors.Is(err, pgx.ErrNoRows) { // If it's an error other than 'no rows'
		log.Printf("Error checking existing participation for user %s on ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("database error checking participation: %w", err)
	}
	// If err is pgx.ErrNoRows, proceed.

	// 4. Create the participant record
	newParticipant := &models.Participant{
		ID:     uuid.New(),
		UserID: userID,
		RideID: rideID,
		Status: string(models.ParticipantStatusPendingPayment), // Initial status (string)
	}
	insertParticipantQuery := `
		INSERT INTO participants (id, user_id, ride_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertParticipantQuery, // Use tx
		newParticipant.ID, newParticipant.UserID, newParticipant.RideID, newParticipant.Status,
	).Scan(&newParticipant.CreatedAt, &newParticipant.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting participant for user %s on ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("failed to record participation: %w", err)
	}

	// 5. No need to decrement seats anymore, it's calculated.
	//    We might update the ride's updated_at timestamp, but it's already handled by the trigger.

	// --- Transaction Commit ---
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Error committing transaction for joining ride %s by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("failed to finalize joining ride: %w", err)
	}

	log.Printf("User %s successfully joined ride %s (Participant ID: %s). Status: %s", userID, rideID, newParticipant.ID, newParticipant.Status)
	return newParticipant, nil
}

// RideContactInfo defines the structure for returning participant contact details.
type RideContactInfo struct {
	UserID    uuid.UUID `json:"user_id"`
	FirstName *string   `json:"first_name"` // Use pointer to handle potential NULL
	LastName  *string   `json:"last_name"`  // Use pointer to handle potential NULL
	WhatsApp  string    `json:"whatsapp"`
	IsCreator bool      `json:"is_creator"` // Flag to indicate if this user created the ride
}

// GetRideContacts retrieves the contact information (WhatsApp) for confirmed participants of a ride.
// It first verifies that the requesting user is also a confirmed participant or the creator.
func (s *RideService) GetRideContacts(ctx context.Context, rideID uuid.UUID, requestingUserID uuid.UUID) ([]RideContactInfo, error) {
	log.Printf("User %s requesting contacts for ride %s", requestingUserID, rideID)

	// 1. Verify the requesting user's status for this ride (must be creator or confirmed participant)
	var requesterStatusStr *string // Use pointer to string to handle NULL/non-enum values
	var isCreator bool
	checkRequesterQuery := `
		SELECT
			p.status, -- Select status directly, will be NULL if no participation record
			(r.user_id = $1) AS is_creator -- Check if requester is the creator
		FROM rides r
		LEFT JOIN participants p ON r.id = p.ride_id AND p.user_id = $1
		WHERE r.id = $2
	`
	err := s.db.QueryRow(ctx, checkRequesterQuery, requestingUserID, rideID).Scan(&requesterStatusStr, &isCreator)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This means the ride itself doesn't exist
			log.Printf("GetRideContacts failed: Ride %s not found.", rideID)
			return nil, errors.New("ride not found")
		}
		// Other potential scan errors
		log.Printf("Error checking requester status for user %s on ride %s: %v", requestingUserID, rideID, err)
		return nil, fmt.Errorf("database error verifying access: %w", err)
	}

	// Convert status string pointer to ParticipantStatus type if not nil
	var requesterStatus models.ParticipantStatus = "not_participant" // Default if not found
	if requesterStatusStr != nil {
		requesterStatus = models.ParticipantStatus(*requesterStatusStr)
		// Optional: Add validation here to ensure the string is a valid status
		// switch requesterStatus {
		// case models.ParticipantStatusConfirmed, models.ParticipantStatusPendingPayment, ...:
		//     // Valid status
		// default:
		//     log.Printf("Warning: Invalid participant status '%s' found in DB for user %s, ride %s", *requesterStatusStr, requestingUserID, rideID)
		//     // Handle appropriately, maybe treat as unauthorized
		//     requesterStatus = "invalid_status"
		// }
	}

	// Check authorization: User must be creator or an active participant
	if !isCreator && requesterStatus != models.ParticipantStatusActive {
		log.Printf("GetRideContacts failed: User %s is not authorized (Status: %s, IsCreator: %t) for ride %s",
			requestingUserID, requesterStatus, isCreator, rideID)
		return nil, errors.New("unauthorized to view contacts for this ride")
	}

	log.Printf("User %s authorized to view contacts for ride %s (Status: %s, IsCreator: %t)",
		requestingUserID, rideID, requesterStatus, isCreator)

	// 2. Fetch confirmed participants and the creator's details
	contacts := []RideContactInfo{}
	getContactsQuery := `
		SELECT
			u.id,
			u.first_name,
			u.last_name,
			u.whatsapp,
			(r.user_id = u.id) AS is_creator
		FROM users u
		JOIN rides r ON r.id = $1 -- Join rides to check creator status easily
		LEFT JOIN participants p ON p.user_id = u.id AND p.ride_id = r.id
		WHERE
			r.id = $1 -- Ensure we only get users related to this ride
			AND (
				r.user_id = u.id -- Include the creator
				OR p.status = $2 -- Include active participants
			)
		GROUP BY u.id, u.first_name, u.last_name, u.whatsapp, r.user_id -- Group to avoid duplicates if user is both creator and participant (shouldn't happen with checks)
	`

	rows, err := s.db.Query(ctx, getContactsQuery, rideID, string(models.ParticipantStatusActive)) // Use Active status
	if err != nil {
		log.Printf("Error fetching contacts for ride %s: %v", rideID, err)
		return nil, fmt.Errorf("database error fetching contacts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var contact RideContactInfo
		err := rows.Scan(
			&contact.UserID,
			&contact.FirstName, // Scan directly into pointers
			&contact.LastName,  // Scan directly into pointers
			&contact.WhatsApp,
			&contact.IsCreator,
		)
		if err != nil {
			log.Printf("Error scanning contact row for ride %s: %v", rideID, err)
			return nil, fmt.Errorf("error processing contact data: %w", err)
		}
		contacts = append(contacts, contact)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating contact rows for ride %s: %v", rideID, err)
		return nil, fmt.Errorf("database iteration error for contacts: %w", err)
	}

	// Security check: Ensure the requesting user is actually in the returned list
	// (Should always be true if authorization check passed, but good for defense)
	foundRequester := false
	for _, c := range contacts {
		if c.UserID == requestingUserID {
			foundRequester = true
			break
		}
	}
	if !foundRequester {
		// This indicates a potential logic error or race condition if status changed between checks
		log.Printf("GetRideContacts Warning: Requesting user %s was authorized but not found in the final contact list for ride %s.", requestingUserID, rideID)
		// Return empty list or error? Returning empty might be safer.
		return []RideContactInfo{}, nil // Return empty list
	}

	log.Printf("Fetched %d contacts for ride %s for user %s", len(contacts), rideID, requestingUserID)
	return contacts, nil
}

// --- New V2 Functions ---

// SearchRides retrieves a list of active rides based on optional search criteria.
func (s *RideService) SearchRides(ctx context.Context, params models.SearchRidesRequest) ([]models.Ride, error) {
	rides := []models.Ride{}
	args := []interface{}{}
	argID := 1

	// Base query selects active rides in the future
	query := `
		SELECT id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status, created_at, updated_at
		FROM rides
		WHERE status = $` + fmt.Sprintf("%d", argID) + `
		AND (departure_date > current_date OR (departure_date = current_date AND departure_time > current_time))
	`
	args = append(args, string(models.RideStatusActive))
	argID++

	// Add optional filters
	if params.StartLocation != nil && *params.StartLocation != "" {
		query += ` AND start_location ILIKE $` + fmt.Sprintf("%d", argID)
		// Use ILIKE for case-insensitive partial matching
		args = append(args, "%"+*params.StartLocation+"%")
		argID++
	}
	if params.EndLocation != nil && *params.EndLocation != "" {
		query += ` AND end_location ILIKE $` + fmt.Sprintf("%d", argID)
		args = append(args, "%"+*params.EndLocation+"%")
		argID++
	}
	if params.DepartureDate != nil && *params.DepartureDate != "" {
		// Ensure date format is validated by handler/model tag before using here
		query += ` AND departure_date = $` + fmt.Sprintf("%d", argID)
		args = append(args, *params.DepartureDate)
		argID++
	}

	// Add ordering
	query += ` ORDER BY departure_date ASC, departure_time ASC`

	// TODO: Add pagination (LIMIT and OFFSET) based on params

	log.Printf("Executing ride search query: %s with args: %v", query, args)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		log.Printf("Error searching rides: %v", err)
		return nil, fmt.Errorf("database error searching rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ride models.Ride
		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
			&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats,
			&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning search result row: %v", err)
			return nil, fmt.Errorf("error processing search results: %w", err)
		}
		rides = append(rides, ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating search results: %v", err)
		return nil, fmt.Errorf("database iteration error during search: %w", err)
	}

	log.Printf("Found %d rides matching search criteria", len(rides))
	return rides, nil
}

// ListUserCreatedRides retrieves rides created by a specific user.
// TODO: Add filtering by status (active/archived/cancelled) and pagination.
func (s *RideService) ListUserCreatedRides(ctx context.Context, userID uuid.UUID) ([]models.Ride, error) {
	rides := []models.Ride{}
	query := `
		SELECT id, user_id, start_location, end_location, departure_date, departure_time, total_seats, status, created_at, updated_at
		FROM rides
		WHERE user_id = $1
		ORDER BY departure_date DESC, departure_time DESC -- Show most recent first
	`
	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		log.Printf("Error fetching rides created by user %s: %v", userID, err)
		return nil, fmt.Errorf("database error fetching created rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ride models.Ride
		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
			&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats,
			&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning created ride row for user %s: %v", userID, err)
			return nil, fmt.Errorf("error processing created ride data: %w", err)
		}
		rides = append(rides, ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating created ride rows for user %s: %v", userID, err)
		return nil, fmt.Errorf("database iteration error for created rides: %w", err)
	}

	log.Printf("Fetched %d rides created by user %s", len(rides), userID)
	return rides, nil
}

// ListUserJoinedRides retrieves rides joined by a specific user (where participation is active).
// TODO: Add filtering by ride status (active/archived/cancelled) and pagination.
func (s *RideService) ListUserJoinedRides(ctx context.Context, userID uuid.UUID) ([]models.Ride, error) {
	rides := []models.Ride{}
	query := `
		SELECT r.id, r.user_id, r.start_location, r.end_location, r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at
		FROM rides r
		JOIN participants p ON r.id = p.ride_id
		WHERE p.user_id = $1 AND p.status = $2 -- Only active participations
		ORDER BY r.departure_date DESC, r.departure_time DESC -- Show most recent first
	`
	rows, err := s.db.Query(ctx, query, userID, string(models.ParticipantStatusActive))
	if err != nil {
		log.Printf("Error fetching rides joined by user %s: %v", userID, err)
		return nil, fmt.Errorf("database error fetching joined rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ride models.Ride
		err := rows.Scan(
			&ride.ID, &ride.UserID, &ride.StartLocation, &ride.EndLocation,
			&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats,
			&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning joined ride row for user %s: %v", userID, err)
			return nil, fmt.Errorf("error processing joined ride data: %w", err)
		}
		rides = append(rides, ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating joined ride rows for user %s: %v", userID, err)
		return nil, fmt.Errorf("database iteration error for joined rides: %w", err)
	}

	log.Printf("Fetched %d rides joined by user %s", len(rides), userID)
	return rides, nil
}

// DeleteRide handles the logic for deleting or cancelling a ride created by the user.
func (s *RideService) DeleteRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (bool, error) {
	// Use transaction for consistency
	pool, ok := s.db.(*pgxpool.Pool)
	if !ok {
		return false, errors.New("database does not support transactions required for DeleteRide")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Error starting transaction for deleting ride %s by user %s: %v", rideID, userID, err)
		return false, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Verify the user is the creator of the ride
	var creatorID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT user_id FROM rides WHERE id = $1`, rideID).Scan(&creatorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("DeleteRide failed: Ride %s not found.", rideID)
			return false, errors.New("ride not found")
		}
		log.Printf("Error verifying ride creator for ride %s: %v", rideID, err)
		return false, fmt.Errorf("database error verifying ride ownership: %w", err)
	}
	if creatorID != userID {
		log.Printf("DeleteRide failed: User %s is not the creator of ride %s", userID, rideID)
		return false, errors.New("unauthorized to delete this ride")
	}

	// 2. Check if there are any active participants
	var activeParticipantCount int
	countQuery := `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND status = $2`
	err = tx.QueryRow(ctx, countQuery, rideID, string(models.ParticipantStatusActive)).Scan(&activeParticipantCount)
	if err != nil {
		log.Printf("Error counting active participants for ride %s during delete: %v", rideID, err)
		return false, fmt.Errorf("database error checking participants: %w", err)
	}

	hasParticipants := activeParticipantCount > 0
	log.Printf("Ride %s has %d active participants. Deletion requested by creator %s.", rideID, activeParticipantCount, userID)

	// 3. Perform action based on participants
	if !hasParticipants {
		// No active participants, perform hard delete
		deleteQuery := `DELETE FROM rides WHERE id = $1`
		_, err = tx.Exec(ctx, deleteQuery, rideID)
		if err != nil {
			log.Printf("Error hard deleting ride %s: %v", rideID, err)
			return false, fmt.Errorf("failed to delete ride: %w", err)
		}
		log.Printf("Ride %s hard deleted successfully.", rideID)
	} else {
		// Has participants, perform soft delete (cancel)
		// a) Update ride status to 'cancelled'
		updateRideQuery := `UPDATE rides SET status = $1, updated_at = NOW() WHERE id = $2`
		_, err = tx.Exec(ctx, updateRideQuery, string(models.RideStatusCancelled), rideID)
		if err != nil {
			log.Printf("Error cancelling ride %s: %v", rideID, err)
			return false, fmt.Errorf("failed to cancel ride: %w", err)
		}
		// b) Update status of active/pending participants to 'cancelled_ride'
		updateParticipantsQuery := `UPDATE participants SET status = $1, updated_at = NOW() WHERE ride_id = $2 AND status IN ($3, $4)`
		_, err = tx.Exec(ctx, updateParticipantsQuery, string(models.ParticipantStatusCancelledRide), rideID, string(models.ParticipantStatusActive), string(models.ParticipantStatusPendingPayment))
		if err != nil {
			log.Printf("Error updating participants status for cancelled ride %s: %v", rideID, err)
			return false, fmt.Errorf("failed to update participants for cancelled ride: %w", err)
		}
		log.Printf("Ride %s cancelled successfully (had participants).", rideID)
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Error committing transaction for deleting/cancelling ride %s: %v", rideID, err)
		return false, fmt.Errorf("failed to finalize ride deletion/cancellation: %w", err)
	}

	return hasParticipants, nil // Return true if participants existed (for warning message)
}

// LeaveRide allows a participant to leave a ride they have joined.
func (s *RideService) LeaveRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) error {
	// Update participant status to 'left'
	// Only allow leaving if status is 'active' or 'pending_payment'? Assume yes for now.
	updateQuery := `
		UPDATE participants
		SET status = $1, updated_at = NOW()
		WHERE ride_id = $2 AND user_id = $3 AND status IN ($4, $5)
	`
	tag, err := s.db.Exec(ctx, updateQuery,
		string(models.ParticipantStatusLeft),
		rideID,
		userID,
		string(models.ParticipantStatusActive),
		string(models.ParticipantStatusPendingPayment),
	)

	if err != nil {
		log.Printf("Error leaving ride %s for user %s: %v", rideID, userID, err)
		return fmt.Errorf("database error leaving ride: %w", err)
	}

	if tag.RowsAffected() == 0 {
		log.Printf("LeaveRide failed: User %s not found as an active/pending participant in ride %s, or already left.", userID, rideID)
		// Check if the ride/user exists to give a more specific error?
		// For now, return a generic error indicating the action couldn't be performed.
		return errors.New("could not leave ride: participation not found or already left/cancelled")
	}

	log.Printf("User %s successfully left ride %s.", userID, rideID)
	return nil
}

// TODO: Add function to archive rides whose departure date/time is in the past.
// This could be called periodically by a background job or triggered via an API endpoint.
// func (s *RideService) ArchivePastRides(ctx context.Context) (int64, error) { ... }
