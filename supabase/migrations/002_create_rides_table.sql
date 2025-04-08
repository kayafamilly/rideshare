-- Migration: 002_create_rides_table
-- Description: Creates the rides table to store ride details.
-- Created at: NOW() [Timestamp will be added by Supabase or manually]

-- Define an ENUM type for ride status for better data integrity
CREATE TYPE ride_status AS ENUM ('open', 'closed', 'full', 'cancelled');

-- Create the rides table
CREATE TABLE rides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),         -- Primary key, auto-generated UUID
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Foreign key referencing the user who created the ride
    start_location TEXT NOT NULL,                           -- Starting point of the ride (e.g., address or place name)
    end_location TEXT NOT NULL,                             -- Destination of the ride (e.g., address or place name)
    departure_date DATE NOT NULL,                           -- Date of the ride
    departure_time TIME NOT NULL,                           -- Time of the ride
    available_seats INTEGER NOT NULL CHECK (available_seats >= 1 AND available_seats <= 5), -- Number of seats offered (1-5)
    status ride_status DEFAULT 'open' NOT NULL,            -- Status of the ride (open, closed, full, cancelled)
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,          -- Timestamp when the ride was created
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL           -- Timestamp when the ride was last updated
);

-- Add comments to the table and columns
COMMENT ON TABLE rides IS 'Stores information about available rides created by users.';
COMMENT ON COLUMN rides.id IS 'Unique identifier for the ride (UUID)';
COMMENT ON COLUMN rides.user_id IS 'Identifier of the user who created the ride (references users.id)';
COMMENT ON COLUMN rides.start_location IS 'Starting location description';
COMMENT ON COLUMN rides.end_location IS 'Ending location description';
COMMENT ON COLUMN rides.departure_date IS 'Date of departure';
COMMENT ON COLUMN rides.departure_time IS 'Time of departure';
COMMENT ON COLUMN rides.available_seats IS 'Number of seats available for participants (1-5)';
COMMENT ON COLUMN rides.status IS 'Current status of the ride (open, closed, full, cancelled)';
COMMENT ON COLUMN rides.created_at IS 'Timestamp of ride creation';
COMMENT ON COLUMN rides.updated_at IS 'Timestamp of last ride update';

-- Create indexes for faster querying
CREATE INDEX idx_rides_user_id ON rides(user_id);
CREATE INDEX idx_rides_departure_date ON rides(departure_date);
CREATE INDEX idx_rides_status ON rides(status);
CREATE INDEX idx_rides_start_location ON rides(start_location); -- Consider using PostGIS for real geo-queries later
CREATE INDEX idx_rides_end_location ON rides(end_location);   -- Consider using PostGIS for real geo-queries later


-- Trigger to automatically update updated_at on rides table update
-- (Uses the function created in the previous migration)
CREATE TRIGGER update_rides_updated_at
BEFORE UPDATE ON rides
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();