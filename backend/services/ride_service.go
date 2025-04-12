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
}

// NewRideService creates a new RideService instance.
func NewRideService(db database.DBPool) *RideService {
	return &RideService{
		validator: validator.New(),
		db:        db,
	}
}

// CreateRide handles the creation of a new ride.
func (s *RideService) CreateRide(ctx context.Context, req models.CreateRideRequest, userID uuid.UUID) (*models.Ride, error) {
	// 1. Validate request data
	if err := s.validator.Struct(req); err != nil {
		log.Printf("Validation error creating ride for user %s: %v", userID, err)
		return nil, fmt.Errorf("invalid ride data: %w", err)
	}
	// Ensure coordinates are provided in the request
	if req.DepartureCoords == nil || req.ArrivalCoords == nil {
		log.Printf("Error creating ride for user %s: Departure or Arrival coordinates are missing in request", userID)
		return nil, errors.New("departure or arrival coordinates are required")
	}

	// 2. Parse date and time strings
	departureDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		log.Printf("Error parsing departure date '%s' for user %s: %v", req.DepartureDate, userID, err)
		return nil, fmt.Errorf("invalid departure date format (use YYYY-MM-DD): %w", err)
	}

	// 3. Validate departure time is in the future
	layout := "2006-01-02 15:04"
	departureDateTimeStr := fmt.Sprintf("%s %s", req.DepartureDate, req.DepartureTime)
	departureDateTime, err := time.Parse(layout, departureDateTimeStr)
	if err != nil {
		log.Printf("Error combining departure date and time '%s %s' for user %s: %v", req.DepartureDate, req.DepartureTime, userID, err)
		return nil, fmt.Errorf("invalid departure date or time format: %w", err)
	}
	if departureDateTime.Before(time.Now()) {
		log.Printf("Validation error: Departure date/time %s is in the past for user %s", departureDateTime, userID)
		return nil, errors.New("departure date and time must be in the future")
	}

	// 4. Create the ride in the database
	newRide := &models.Ride{
		ID:                    uuid.New(),
		UserID:                userID,
		DepartureLocationName: req.DepartureLocationName,
		DepartureCoords:       req.DepartureCoords,
		ArrivalLocationName:   req.ArrivalLocationName,
		ArrivalCoords:         req.ArrivalCoords,
		DepartureDate:         departureDate,
		DepartureTime:         req.DepartureTime,
		TotalSeats:            req.TotalSeats,
		Status:                string(models.RideStatusActive),
	}

	// Use ST_SetSRID(ST_MakePoint(longitude, latitude), 4326) for inserting coordinates
	insertQuery := `
		INSERT INTO rides (
			id, user_id,
			departure_location_name, departure_coords,
			arrival_location_name, arrival_coords,
			departure_date, departure_time, total_seats, status
		)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326), $6, ST_SetSRID(ST_MakePoint($7, $8), 4326), $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	err = s.db.QueryRow(ctx, insertQuery,
		newRide.ID, newRide.UserID,
		newRide.DepartureLocationName, newRide.DepartureCoords.Longitude, newRide.DepartureCoords.Latitude, // Lon, Lat for departure
		newRide.ArrivalLocationName, newRide.ArrivalCoords.Longitude, newRide.ArrivalCoords.Latitude, // Lon, Lat for arrival
		newRide.DepartureDate, newRide.DepartureTime, newRide.TotalSeats, newRide.Status,
	).Scan(&newRide.CreatedAt, &newRide.UpdatedAt)

	if err != nil {
		log.Printf("Error inserting new ride for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to create ride in database: %w", err)
	}

	log.Printf("Ride created successfully by user %s: Ride ID %s", userID, newRide.ID)
	return newRide, nil
}

// scanRideRow scans a row from a rides query into a models.Ride struct, handling coordinates.
func scanRideRow(rows pgx.Row) (*models.Ride, error) {
	var ride models.Ride
	var depLon, depLat, arrLon, arrLat *float64 // Use pointers to handle potential NULLs from LEFT JOINs or if coords aren't selected

	// Adjust scan arguments based on the specific query's SELECT list
	// This example assumes all standard fields + coordinates + creator name + places taken are selected
	err := rows.Scan(
		&ride.ID, &ride.UserID,
		&ride.DepartureLocationName, &depLon, &depLat,
		&ride.ArrivalLocationName, &arrLon, &arrLat,
		&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats,
		&ride.Status, &ride.CreatedAt, &ride.UpdatedAt,
		&ride.PlacesTaken,      // Assumes this is calculated/selected in the query
		&ride.CreatorFirstName, // Assumes this is joined/selected in the query
	)
	if err != nil {
		return nil, err // Return scan error directly
	}

	// Populate GeoPoint structs if coordinates were scanned successfully
	if depLon != nil && depLat != nil {
		ride.DepartureCoords = &models.GeoPoint{Longitude: *depLon, Latitude: *depLat}
	}
	if arrLon != nil && arrLat != nil {
		ride.ArrivalCoords = &models.GeoPoint{Longitude: *arrLon, Latitude: *arrLat}
	}
	return &ride, nil
}

// scanRideRowBasic scans a row with basic ride details + coordinates + creator name
func scanRideRowBasic(row pgx.Row) (*models.Ride, error) {
	var ride models.Ride
	var depLon, depLat, arrLon, arrLat *float64

	err := row.Scan(
		&ride.ID, &ride.UserID,
		&ride.DepartureLocationName, &depLon, &depLat,
		&ride.ArrivalLocationName, &arrLon, &arrLat,
		&ride.DepartureDate, &ride.DepartureTime, &ride.TotalSeats,
		&ride.Status,
		&ride.CreatedAt, &ride.UpdatedAt,
		&ride.CreatorFirstName, // Assumes creator name is joined
	)
	if err != nil {
		return nil, err
	}

	if depLon != nil && depLat != nil {
		ride.DepartureCoords = &models.GeoPoint{Longitude: *depLon, Latitude: *depLat}
	}
	if arrLon != nil && arrLat != nil {
		ride.ArrivalCoords = &models.GeoPoint{Longitude: *arrLon, Latitude: *arrLat}
	}
	return &ride, nil
}

// ListAvailableRides retrieves a list of rides that are currently 'active'.
func (s *RideService) ListAvailableRides(ctx context.Context) ([]models.Ride, error) {
	rides := []models.Ride{}
	query := `
		SELECT
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM participants p WHERE p.ride_id = r.id AND p.status = 'active') AS places_taken,
			u.first_name AS creator_first_name
		FROM rides r
		JOIN users u ON r.user_id = u.id
		WHERE r.status = $1
		  AND (r.departure_date > current_date OR (r.departure_date = current_date AND r.departure_time > current_time))
		  AND (SELECT COUNT(*) FROM participants p WHERE p.ride_id = r.id AND p.status = 'active') < r.total_seats
		ORDER BY r.departure_date ASC, r.departure_time ASC
	`

	rows, err := s.db.Query(ctx, query, string(models.RideStatusActive))
	if err != nil {
		log.Printf("Error querying available rides: %v", err)
		return nil, fmt.Errorf("database error fetching rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ride, err := scanRideRow(rows) // Use the helper function
		if err != nil {
			log.Printf("Error scanning available ride row: %v", err)
			return nil, fmt.Errorf("error processing ride data: %w", err)
		}
		rides = append(rides, *ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating available ride rows: %v", err)
		return nil, fmt.Errorf("database iteration error: %w", err)
	}

	log.Printf("Fetched %d available rides", len(rides))
	return rides, nil
}

// GetRideDetails retrieves details for a specific ride by its ID.
func (s *RideService) GetRideDetails(ctx context.Context, rideID uuid.UUID) (*models.Ride, error) {
	query := `
		SELECT
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status,
			r.created_at, r.updated_at,
			u.first_name AS creator_first_name
		FROM rides r
		JOIN users u ON r.user_id = u.id
		WHERE r.id = $1
	`
	ride, err := scanRideRowBasic(s.db.QueryRow(ctx, query, rideID)) // Use basic scanner
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Ride not found: ID %s", rideID)
			return nil, errors.New("ride not found")
		}
		log.Printf("Error fetching ride details for ID %s: %v", rideID, err)
		return nil, fmt.Errorf("database error fetching ride details: %w", err)
	}

	// Calculate places taken separately
	var activeParticipantsCount int
	countQuery := `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND status = $2`
	err = s.db.QueryRow(ctx, countQuery, rideID, string(models.ParticipantStatusActive)).Scan(&activeParticipantsCount)
	if err != nil {
		log.Printf("Error counting active participants for ride %s during GetRideDetails: %v", rideID, err)
		ride.PlacesTaken = 0 // Fallback
	} else {
		ride.PlacesTaken = activeParticipantsCount
	}

	log.Printf("Fetched details for ride ID %s (Places Taken: %d)", rideID, ride.PlacesTaken)
	return ride, nil
}

// JoinRide allows a user to join an existing ride.
func (s *RideService) JoinRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (*models.Participant, error) {
	pool, ok := s.db.(*pgxpool.Pool)
	if !ok {
		log.Println("Warning: Database pool does not support transactions, proceeding without.")
		return nil, errors.New("database does not support transactions required for JoinRide")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("Error starting transaction for joining ride %s by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Get ride details and lock the row (only need fields for validation)
	var ride models.Ride
	lockQuery := `
		SELECT id, user_id, total_seats, status
		FROM rides
		WHERE id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockQuery, rideID).Scan(
		&ride.ID, &ride.UserID, &ride.TotalSeats, &ride.Status,
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
	if ride.Status != string(models.RideStatusActive) {
		log.Printf("JoinRide failed: Ride %s is not active (status: %s)", rideID, ride.Status)
		return nil, errors.New("ride is not active for joining")
	}

	var activeParticipantsCount int
	countQuery := `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND status = $2`
	err = tx.QueryRow(ctx, countQuery, rideID, string(models.ParticipantStatusActive)).Scan(&activeParticipantsCount)
	if err != nil {
		log.Printf("Error counting active participants for ride %s: %v", rideID, err)
		return nil, fmt.Errorf("database error checking ride capacity: %w", err)
	}

	if activeParticipantsCount >= ride.TotalSeats {
		log.Printf("JoinRide failed: Ride %s is full (%d/%d seats taken)", rideID, activeParticipantsCount, ride.TotalSeats)
		return nil, errors.New("ride is already full")
	}
	if ride.UserID == userID {
		log.Printf("JoinRide failed: User %s cannot join their own ride %s", userID, rideID)
		return nil, errors.New("you cannot join your own ride")
	}

	// 3. Check existing participation
	var existingParticipant models.Participant
	checkParticipantQuery := `SELECT id, status FROM participants WHERE user_id = $1 AND ride_id = $2`
	err = tx.QueryRow(ctx, checkParticipantQuery, userID, rideID).Scan(&existingParticipant.ID, &existingParticipant.Status)

	if err == nil { // Record found
		switch existingParticipant.Status {
		case string(models.ParticipantStatusActive), string(models.ParticipantStatusPendingPayment):
			log.Printf("JoinRide failed: User %s has already joined ride %s with status '%s'", userID, rideID, existingParticipant.Status)
			return nil, errors.New("you have already joined this ride or payment is pending")
		case string(models.ParticipantStatusLeft):
			log.Printf("User %s previously left ride %s. Updating status to pending_payment.", userID, rideID)
			updateStatusQuery := `UPDATE participants SET status = $1, updated_at = NOW() WHERE id = $2 RETURNING created_at, updated_at` // Also return timestamps
			updateErr := tx.QueryRow(ctx, updateStatusQuery, string(models.ParticipantStatusPendingPayment), existingParticipant.ID).Scan(&existingParticipant.CreatedAt, &existingParticipant.UpdatedAt)
			if updateErr != nil {
				log.Printf("Error updating status for rejoining participant %s on ride %s: %v", userID, rideID, updateErr)
				return nil, fmt.Errorf("failed to update participation status for rejoin: %w", updateErr)
			}
			existingParticipant.Status = string(models.ParticipantStatusPendingPayment)
			existingParticipant.UserID = userID // Ensure UserID and RideID are set
			existingParticipant.RideID = rideID
			// Commit transaction after successful update
			commitErr := tx.Commit(ctx)
			if commitErr != nil {
				log.Printf("Error committing transaction after rejoining ride %s by user %s: %v", rideID, userID, commitErr)
				return nil, fmt.Errorf("failed to finalize rejoining ride: %w", commitErr)
			}
			return &existingParticipant, nil
		default:
			log.Printf("JoinRide failed: User %s has an unexpected participation status '%s' for ride %s", userID, rideID, existingParticipant.Status)
			return nil, fmt.Errorf("unexpected participation status: %s", existingParticipant.Status)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("Error checking existing participation for user %s on ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("database error checking participation: %w", err)
	}

	// 4. Create NEW participant record
	newParticipant := &models.Participant{
		ID:     uuid.New(),
		UserID: userID,
		RideID: rideID,
		Status: string(models.ParticipantStatusPendingPayment),
	}
	insertParticipantQuery := `
		INSERT INTO participants (id, user_id, ride_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertParticipantQuery,
		newParticipant.ID, newParticipant.UserID, newParticipant.RideID, newParticipant.Status,
	).Scan(&newParticipant.CreatedAt, &newParticipant.UpdatedAt)
	if err != nil {
		log.Printf("Error inserting participant for user %s on ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("failed to record participation: %w", err)
	}

	// 5. Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Error committing transaction for joining ride %s by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("failed to finalize joining ride: %w", err)
	}

	log.Printf("User %s successfully joined ride %s (Participant ID: %s). Status: %s", userID, rideID, newParticipant.ID, newParticipant.Status)
	return newParticipant, nil
}

// ValidateRideForJoiningTx performs validation checks within an existing transaction.
func (s *RideService) ValidateRideForJoiningTx(ctx context.Context, tx pgx.Tx, rideID uuid.UUID, userID uuid.UUID) (*models.Ride, error) {
	var ride models.Ride
	// Only select fields needed for validation
	lockQuery := `
		SELECT id, user_id, total_seats, status
		FROM rides
		WHERE id = $1
		FOR UPDATE
	`
	err := tx.QueryRow(ctx, lockQuery, rideID).Scan(
		&ride.ID, &ride.UserID, &ride.TotalSeats, &ride.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("ValidationTx failed: Ride not found: ID %s", rideID)
			return nil, errors.New("ride not found")
		}
		log.Printf("Error fetching/locking ride %s for validation by user %s: %v", rideID, userID, err)
		return nil, fmt.Errorf("database error fetching ride for validation: %w", err)
	}

	if ride.Status != string(models.RideStatusActive) {
		log.Printf("ValidationTx failed: Ride %s is not active (status: %s)", rideID, ride.Status)
		return nil, errors.New("ride is not open for joining")
	}

	var activeParticipantsCount int
	countQuery := `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND status = $2`
	err = tx.QueryRow(ctx, countQuery, rideID, string(models.ParticipantStatusActive)).Scan(&activeParticipantsCount)
	if err != nil {
		log.Printf("Error counting active participants for ride %s during validation: %v", rideID, err)
		return nil, fmt.Errorf("database error checking ride capacity: %w", err)
	}

	if activeParticipantsCount >= ride.TotalSeats {
		log.Printf("ValidationTx failed: Ride %s is full (%d/%d seats taken)", rideID, activeParticipantsCount, ride.TotalSeats)
		return nil, errors.New("ride is already full")
	}

	if ride.UserID == userID {
		log.Printf("ValidationTx failed: User %s cannot join their own ride %s", userID, rideID)
		return nil, errors.New("you cannot join your own ride")
	}

	var participationCount int
	countQuery = `SELECT COUNT(*) FROM participants WHERE ride_id = $1 AND user_id = $2 AND status = $3`
	err = tx.QueryRow(ctx, countQuery, rideID, userID, string(models.ParticipantStatusActive)).Scan(&participationCount)
	if err != nil {
		log.Printf("Error checking active participation for user %s on ride %s: %v", userID, rideID, err)
		return nil, fmt.Errorf("database error checking participation: %w", err)
	}
	if participationCount > 0 {
		log.Printf("ValidationTx failed: User %s has already actively joined ride %s", userID, rideID)
		return nil, errors.New("you have already joined this ride")
	}

	log.Printf("ValidationTx successful for user %s joining ride %s", userID, rideID)
	return &ride, nil
}

// RideContactInfo defines the structure for returning participant contact details.
type RideContactInfo struct {
	UserID    uuid.UUID `json:"user_id"`
	FirstName *string   `json:"first_name"`
	LastName  *string   `json:"last_name"`
	WhatsApp  string    `json:"whatsapp"`
	IsCreator bool      `json:"is_creator"`
}

// GetRideContacts retrieves contact info for confirmed participants and the creator.
func (s *RideService) GetRideContacts(ctx context.Context, rideID uuid.UUID, requestingUserID uuid.UUID) ([]RideContactInfo, error) {
	log.Printf("User %s requesting contacts for ride %s", requestingUserID, rideID)

	var requesterStatusStr *string
	var isCreator bool
	checkRequesterQuery := `
		SELECT
			p.status,
			(r.user_id = $1) AS is_creator
		FROM rides r
		LEFT JOIN participants p ON r.id = p.ride_id AND p.user_id = $1
		WHERE r.id = $2
	`
	err := s.db.QueryRow(ctx, checkRequesterQuery, requestingUserID, rideID).Scan(&requesterStatusStr, &isCreator)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("GetRideContacts failed: Ride %s not found.", rideID)
			return nil, errors.New("ride not found")
		}
		log.Printf("Error checking requester status for user %s on ride %s: %v", requestingUserID, rideID, err)
		return nil, fmt.Errorf("database error verifying access: %w", err)
	}

	var requesterStatus models.ParticipantStatus = "not_participant"
	if requesterStatusStr != nil {
		requesterStatus = models.ParticipantStatus(*requesterStatusStr)
	}

	if !isCreator && requesterStatus != models.ParticipantStatusActive {
		log.Printf("GetRideContacts failed: User %s is not authorized (Status: %s, IsCreator: %t) for ride %s",
			requestingUserID, requesterStatus, isCreator, rideID)
		return nil, errors.New("unauthorized to view contacts for this ride")
	}

	log.Printf("User %s authorized to view contacts for ride %s (Status: %s, IsCreator: %t)",
		requestingUserID, rideID, requesterStatus, isCreator)

	contacts := []RideContactInfo{}
	getContactsQuery := `
		SELECT
			u.id, u.first_name, u.last_name, u.whatsapp,
			(r.user_id = u.id) AS is_creator
		FROM users u
		JOIN rides r ON r.id = $1
		LEFT JOIN participants p ON p.user_id = u.id AND p.ride_id = r.id
		WHERE
			r.id = $1
			AND (
				r.user_id = u.id -- Include the creator OR
				OR p.status = $2 -- Include active participants
			)
			AND u.deleted_at IS NULL -- Exclude deleted users
	`
	rows, err := s.db.Query(ctx, getContactsQuery, rideID, string(models.ParticipantStatusActive))
	if err != nil {
		log.Printf("Error fetching contacts for ride %s: %v", rideID, err)
		return nil, fmt.Errorf("database error fetching contacts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var contact RideContactInfo
		err := rows.Scan(
			&contact.UserID, &contact.FirstName, &contact.LastName, &contact.WhatsApp, &contact.IsCreator,
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

	log.Printf("Fetched %d contacts for ride %s", len(contacts), rideID)
	return contacts, nil
}

// SearchRides searches for available rides based on criteria.
func (s *RideService) SearchRides(ctx context.Context, params models.SearchRidesRequest) ([]models.Ride, error) {
	// 1. Validate parameters (basic validation done via tags, add more if needed)
	if err := s.validator.Struct(params); err != nil {
		log.Printf("Validation error during ride search: %v", err)
		return nil, fmt.Errorf("invalid search parameters: %w", err)
	}

	// 2. Build the base query
	baseQuery := `
		SELECT
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM participants p WHERE p.ride_id = r.id AND p.status = 'active') AS places_taken,
			u.first_name AS creator_first_name
		FROM rides r
		JOIN users u ON r.user_id = u.id
		WHERE r.status = $1 -- Always filter for active rides
		  AND (r.departure_date > current_date OR (r.departure_date = current_date AND r.departure_time > current_time)) -- Filter out past rides
		  AND (SELECT COUNT(*) FROM participants p WHERE p.ride_id = r.id AND p.status = 'active') < r.total_seats -- Filter out full rides
	`
	args := []interface{}{string(models.RideStatusActive)}
	argID := 2 // Start next argument index at 2

	// 3. Add filters dynamically
	if params.StartLocation != nil && *params.StartLocation != "" {
		baseQuery += fmt.Sprintf(" AND r.departure_location_name ILIKE $%d", argID)
		args = append(args, "%"+*params.StartLocation+"%") // Use LIKE for partial matching
		argID++
	}
	if params.EndLocation != nil && *params.EndLocation != "" {
		baseQuery += fmt.Sprintf(" AND r.arrival_location_name ILIKE $%d", argID)
		args = append(args, "%"+*params.EndLocation+"%")
		argID++
	}
	if params.DepartureDate != nil && *params.DepartureDate != "" {
		baseQuery += fmt.Sprintf(" AND r.departure_date = $%d", argID)
		args = append(args, *params.DepartureDate)
		argID++
	}

	// 4. Add ordering
	baseQuery += " ORDER BY r.departure_date ASC, r.departure_time ASC"

	// 5. Add pagination
	limit := 20 // Default limit
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}
	baseQuery += fmt.Sprintf(" LIMIT $%d", argID)
	args = append(args, limit)
	argID++

	offset := 0 // Default offset
	if params.Page != nil && *params.Page > 1 {
		offset = (*params.Page - 1) * limit
	}
	baseQuery += fmt.Sprintf(" OFFSET $%d", argID)
	args = append(args, offset)
	// argID++ // No need to increment further

	log.Printf("Executing ride search query: %s with args: %v", baseQuery, args)

	// 6. Execute query
	rows, err := s.db.Query(ctx, baseQuery, args...)
	if err != nil {
		log.Printf("Error executing ride search query: %v", err)
		return nil, fmt.Errorf("database error searching rides: %w", err)
	}
	defer rows.Close()

	// 7. Scan results
	rides := []models.Ride{}
	for rows.Next() {
		ride, err := scanRideRow(rows) // Use the helper
		if err != nil {
			log.Printf("Error scanning search result row: %v", err)
			return nil, fmt.Errorf("error processing search result data: %w", err)
		}
		rides = append(rides, *ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating search result rows: %v", err)
		return nil, fmt.Errorf("database iteration error during search: %w", err)
	}

	log.Printf("Found %d rides matching search criteria", len(rides))
	return rides, nil
}

// ListUserCreatedRides retrieves rides created by a specific user.
func (s *RideService) ListUserCreatedRides(ctx context.Context, userID uuid.UUID) ([]models.Ride, error) {
	rides := []models.Ride{}
	query := `
		SELECT
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM participants p WHERE p.ride_id = r.id AND p.status = 'active') AS places_taken,
			u.first_name AS creator_first_name -- Creator name is the user themselves here
		FROM rides r
		JOIN users u ON r.user_id = u.id
		WHERE r.user_id = $1
		ORDER BY r.departure_date DESC, r.departure_time DESC -- Show most recent first
	`
	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		log.Printf("Error querying created rides for user %s: %v", userID, err)
		return nil, fmt.Errorf("database error fetching created rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ride, err := scanRideRow(rows) // Use helper
		if err != nil {
			log.Printf("Error scanning created ride row for user %s: %v", userID, err)
			return nil, fmt.Errorf("error processing created ride data: %w", err)
		}
		rides = append(rides, *ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating created ride rows for user %s: %v", userID, err)
		return nil, fmt.Errorf("database iteration error for created rides: %w", err)
	}

	log.Printf("Fetched %d created rides for user %s", len(rides), userID)
	return rides, nil
}

// ListUserJoinedRides retrieves rides a specific user has joined (and is active).
func (s *RideService) ListUserJoinedRides(ctx context.Context, userID uuid.UUID) ([]models.Ride, error) {
	rides := []models.Ride{}
	query := `
		SELECT
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM participants p_count WHERE p_count.ride_id = r.id AND p_count.status = 'active') AS places_taken,
			u.first_name AS creator_first_name
		FROM rides r
		JOIN participants p ON r.id = p.ride_id
		JOIN users u ON r.user_id = u.id -- Join users table for creator info
		WHERE p.user_id = $1 AND p.status = $2 -- Filter by user ID and active participation status
		ORDER BY r.departure_date ASC, r.departure_time ASC -- Show upcoming first
	`
	rows, err := s.db.Query(ctx, query, userID, string(models.ParticipantStatusActive))
	if err != nil {
		log.Printf("Error querying joined rides for user %s: %v", userID, err)
		return nil, fmt.Errorf("database error fetching joined rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ride, err := scanRideRow(rows) // Use helper
		if err != nil {
			log.Printf("Error scanning joined ride row for user %s: %v", userID, err)
			return nil, fmt.Errorf("error processing joined ride data: %w", err)
		}
		rides = append(rides, *ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating joined ride rows for user %s: %v", userID, err)
		return nil, fmt.Errorf("database iteration error for joined rides: %w", err)
	}

	log.Printf("Fetched %d joined rides for user %s", len(rides), userID)
	return rides, nil
}

// DeleteRide handles deleting a ride, checking permissions first.
func (s *RideService) DeleteRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (bool, error) {
	log.Printf("User %s attempting to delete ride %s", userID, rideID)

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

	// 1. Verify ownership and get participant count
	var rideUserID uuid.UUID
	var participantCount int
	checkQuery := `
		SELECT r.user_id, COUNT(p.id)
		FROM rides r
		LEFT JOIN participants p ON r.id = p.ride_id AND p.status = 'active'
		WHERE r.id = $1
		GROUP BY r.user_id
	`
	err = tx.QueryRow(ctx, checkQuery, rideID).Scan(&rideUserID, &participantCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("DeleteRide failed: Ride %s not found.", rideID)
			return false, errors.New("ride not found")
		}
		log.Printf("Error checking ride ownership/participants for ride %s: %v", rideID, err)
		return false, fmt.Errorf("database error checking ride details: %w", err)
	}

	// 2. Check ownership
	if rideUserID != userID {
		log.Printf("DeleteRide failed: User %s does not own ride %s", userID, rideID)
		return false, errors.New("unauthorized to delete this ride")
	}

	// 3. Perform deletion (soft or hard based on participants)
	// For now, let's implement a hard delete as per original logic (no participants check mentioned in v2 for delete)
	// If soft delete is preferred, update status to 'cancelled' or similar.
	// Let's stick to hard delete for now. We need to delete participants first due to FK constraints.
	deleteParticipantsQuery := `DELETE FROM participants WHERE ride_id = $1`
	_, err = tx.Exec(ctx, deleteParticipantsQuery, rideID)
	if err != nil {
		log.Printf("Error deleting participants for ride %s during ride deletion: %v", rideID, err)
		return false, fmt.Errorf("failed to delete ride participants: %w", err)
	}

	deleteRideQuery := `DELETE FROM rides WHERE id = $1 AND user_id = $2`
	tag, err := tx.Exec(ctx, deleteRideQuery, rideID, userID)
	if err != nil {
		log.Printf("Error deleting ride %s owned by user %s: %v", rideID, userID, err)
		return false, fmt.Errorf("failed to delete ride: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Should not happen if ownership check passed, but handle defensively
		log.Printf("DeleteRide failed: Ride %s not found or ownership mismatch after check.", rideID)
		return false, errors.New("ride not found or could not be deleted")
	}

	// 4. Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Error committing transaction for deleting ride %s: %v", rideID, err)
		return false, fmt.Errorf("failed to finalize ride deletion: %w", err)
	}

	log.Printf("Ride %s deleted successfully by user %s", rideID, userID)
	// Return participantCount > 0 to indicate if participants were present (as per v2 spec popup)
	return participantCount > 0, nil
}

// LeaveRide allows a user to leave a ride they have joined.
func (s *RideService) LeaveRide(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) error {
	log.Printf("User %s attempting to leave ride %s", userID, rideID)

	// Update participant status to 'left'
	// We only allow leaving if the current status is 'active' or 'pending_payment'
	query := `
		UPDATE participants
		SET status = $1, updated_at = NOW()
		WHERE ride_id = $2 AND user_id = $3 AND (status = $4 OR status = $5)
	`
	tag, err := s.db.Exec(ctx, query,
		string(models.ParticipantStatusLeft),
		rideID,
		userID,
		string(models.ParticipantStatusActive),
		string(models.ParticipantStatusPendingPayment),
	)

	if err != nil {
		log.Printf("Error updating participant status to 'left' for user %s on ride %s: %v", userID, rideID, err)
		return fmt.Errorf("database error leaving ride: %w", err)
	}

	if tag.RowsAffected() == 0 {
		log.Printf("LeaveRide failed: User %s not found as an active/pending participant on ride %s, or ride not found.", userID, rideID)
		// Check if the ride exists at all to give a better error message
		var exists bool
		checkRideQuery := `SELECT EXISTS(SELECT 1 FROM rides WHERE id = $1)`
		_ = s.db.QueryRow(ctx, checkRideQuery, rideID).Scan(&exists)
		if !exists {
			return errors.New("ride not found")
		}
		return errors.New("you are not currently an active participant in this ride")
	}

	log.Printf("User %s successfully left ride %s", userID, rideID)
	// TODO: Consider if any notification should be sent to the creator?
	return nil
}

// GetUserParticipationStatus checks if a user is participating in a ride and returns their status.
func (s *RideService) GetUserParticipationStatus(ctx context.Context, rideID uuid.UUID, userID uuid.UUID) (string, error) {
	var status string
	query := `SELECT status FROM participants WHERE ride_id = $1 AND user_id = $2`
	err := s.db.QueryRow(ctx, query, rideID, userID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "not_participating", nil // User is not in the participants table for this ride
		}
		log.Printf("Error fetching participation status for user %s on ride %s: %v", userID, rideID, err)
		return "", fmt.Errorf("database error fetching participation status: %w", err)
	}
	return status, nil
}

// ListUserHistoryRides retrieves past or cancelled rides for a user (both created and joined).
func (s *RideService) ListUserHistoryRides(ctx context.Context, userID uuid.UUID) ([]models.Ride, error) {
	rides := []models.Ride{}
	// Select rides created by the user OR joined by the user WHERE the ride status is archived/cancelled OR departure is in the past
	query := `
		SELECT DISTINCT -- Use DISTINCT to avoid duplicates if user created AND joined (though joining own ride is disallowed)
			r.id, r.user_id,
			r.departure_location_name, ST_X(r.departure_coords) AS departure_lon, ST_Y(r.departure_coords) AS departure_lat,
			r.arrival_location_name, ST_X(r.arrival_coords) AS arrival_lon, ST_Y(r.arrival_coords) AS arrival_lat,
			r.departure_date, r.departure_time, r.total_seats, r.status, r.created_at, r.updated_at,
			(SELECT COUNT(*) FROM participants p_count WHERE p_count.ride_id = r.id AND p_count.status = 'active') AS places_taken,
			u.first_name AS creator_first_name
		FROM rides r
		JOIN users u ON r.user_id = u.id
		LEFT JOIN participants p ON r.id = p.ride_id AND p.user_id = $1 -- Join participants for the requesting user
		WHERE
			(r.user_id = $1 OR p.user_id = $1) -- Ride created by user OR joined by user
			AND
			(
				r.status = $2 OR r.status = $3 -- Ride is archived or cancelled
				OR (r.departure_date < current_date OR (r.departure_date = current_date AND r.departure_time <= current_time)) -- Ride is in the past
			)
		ORDER BY r.departure_date DESC, r.departure_time DESC
	`
	rows, err := s.db.Query(ctx, query, userID, string(models.RideStatusArchived), string(models.RideStatusCancelled))
	if err != nil {
		log.Printf("Error querying history rides for user %s: %v", userID, err)
		return nil, fmt.Errorf("database error fetching history rides: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ride, err := scanRideRow(rows) // Use helper
		if err != nil {
			log.Printf("Error scanning history ride row for user %s: %v", userID, err)
			return nil, fmt.Errorf("error processing history ride data: %w", err)
		}
		rides = append(rides, *ride)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating history ride rows for user %s: %v", userID, err)
		return nil, fmt.Errorf("database iteration error for history rides: %w", err)
	}

	log.Printf("Fetched %d history rides for user %s", len(rides), userID)
	return rides, nil
}
