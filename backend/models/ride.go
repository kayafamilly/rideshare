package models

import (
	"time"

	"github.com/google/uuid"
)

// RideStatus represents the possible statuses of a ride (now using TEXT in DB).
type RideStatus string

const (
	RideStatusActive    RideStatus = "active"    // Default status, ride is visible and joinable if seats available
	RideStatusArchived  RideStatus = "archived"  // Ride is in the past or manually archived
	RideStatusCancelled RideStatus = "cancelled" // Cancelled by the creator
	// Note: 'full' is not a status anymore, it's determined by calculation (total_seats - active_participants)
)

// GeoPoint represents geographic coordinates.
type GeoPoint struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// Ride represents the structure for the 'rides' table.
type Ride struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	UserID                uuid.UUID `json:"user_id" db:"user_id"`                                 // Creator's User ID
	DepartureLocationName string    `json:"departure_location_name" db:"departure_location_name"` // User-friendly name
	DepartureCoords       *GeoPoint `json:"departure_coords" db:"departure_coords"`               // Geographic coordinates (using custom scan logic or pgtype)
	ArrivalLocationName   string    `json:"arrival_location_name" db:"arrival_location_name"`     // User-friendly name
	ArrivalCoords         *GeoPoint `json:"arrival_coords" db:"arrival_coords"`                   // Geographic coordinates (using custom scan logic or pgtype)
	DepartureDate         time.Time `json:"departure_date" db:"departure_date"`                   // Date of departure
	DepartureTime         string    `json:"departure_time" db:"departure_time"`                   // Time of departure (HH:MM format) - Stored as TIME in DB
	TotalSeats            int       `json:"total_seats" db:"total_seats"`                         // Total seats offered by creator (1-5)
	Status                string    `json:"status" db:"status"`                                   // active, archived, cancelled (now TEXT)
	PlacesTaken           int       `json:"places_taken"`                                         // Calculated field, not directly from DB column 'nb_places_prises'
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
	// Optional: Include creator info when fetching rides
	CreatorFirstName *string `json:"creator_first_name,omitempty" db:"creator_first_name"` // Populated by JOIN in GetRideDetails
}

// ParticipantStatus represents the possible statuses of a participant (now using TEXT in DB).
type ParticipantStatus string

const (
	ParticipantStatusPendingPayment ParticipantStatus = "pending_payment" // User joined, waiting for payment confirmation
	ParticipantStatusActive         ParticipantStatus = "active"          // Payment successful, user is an active participant
	ParticipantStatusLeft           ParticipantStatus = "left"            // User chose to leave the ride
	ParticipantStatusCancelledRide  ParticipantStatus = "cancelled_ride"  // Ride was cancelled by creator after user joined/paid
)

// Participant represents the structure for the 'participants' table.
type Participant struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"` // Participating User ID
	RideID    uuid.UUID `json:"ride_id" db:"ride_id"` // Ride ID being joined
	Status    string    `json:"status" db:"status"`   // Now TEXT
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	// Optional: Include user/ride info when fetching participants
	User *User `json:"user,omitempty" db:"-"` // Participating user info (populated in service)
	Ride *Ride `json:"ride,omitempty" db:"-"` // Ride info (populated in service)
}

// --- DTOs (Data Transfer Objects) for API Requests/Responses ---

// CreateRideRequest defines the structure for creating a new ride, including geographic data.
type CreateRideRequest struct {
	DepartureLocationName string    `json:"departure_location_name" validate:"required"`
	DepartureCoords       *GeoPoint `json:"departure_coords" validate:"required"` // Frontend sends { longitude, latitude }
	ArrivalLocationName   string    `json:"arrival_location_name" validate:"required"`
	ArrivalCoords         *GeoPoint `json:"arrival_coords" validate:"required"`                     // Frontend sends { longitude, latitude }
	DepartureDate         string    `json:"departure_date" validate:"required,datetime=2006-01-02"` // YYYY-MM-DD
	DepartureTime         string    `json:"departure_time" validate:"required,datetime=15:04"`      // HH:MM (24-hour format)
	TotalSeats            int       `json:"total_seats" validate:"required,min=1,max=5"`
}

// RideResponse defines a structure for returning ride details, potentially including creator info.
type RideResponse struct {
	Ride
	CreatorName        string `json:"creator_name,omitempty"`         // Example: Add creator's name directly
	ParticipantsCount  int    `json:"participants_count,omitempty"`   // Number of active participants
	AvailableSeatsLeft int    `json:"available_seats_left,omitempty"` // Calculated: TotalSeats - ParticipantsCount
}

// SearchRidesRequest defines optional query parameters for searching rides.
type SearchRidesRequest struct {
	StartLocation *string `query:"start_location"`                                          // Optional start location filter (e.g., using LIKE %query%)
	EndLocation   *string `query:"end_location"`                                            // Optional end location filter
	DepartureDate *string `query:"departure_date" validate:"omitempty,datetime=2006-01-02"` // Optional date filter (YYYY-MM-DD)
	Page          *int    `query:"page" validate:"omitempty,min=1"`                         // Optional pagination: page number (1-based)
	Limit         *int    `query:"limit" validate:"omitempty,min=1,max=100"`                // Optional pagination: items per page (e.g., 1-100)
}

// JoinRideResponse defines the structure for responding after a user joins a ride.
type JoinRideResponse struct {
	ParticipationID uuid.UUID `json:"participation_id"`
	RideID          uuid.UUID `json:"ride_id"`
	UserID          uuid.UUID `json:"user_id"`
	Status          string    `json:"status"` // Should be 'pending_payment' initially (now TEXT)
	Message         string    `json:"message"`
}

// Note: Updated Ride/Participant statuses to string. Renamed AvailableSeats to TotalSeats.
// Note: Added calculated fields to RideResponse. Added SearchRidesRequest DTO.
