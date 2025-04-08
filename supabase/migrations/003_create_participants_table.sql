-- Migration: 003_create_participants_table
-- Description: Creates the participants table to link users to the rides they join.
-- Created at: NOW() [Timestamp will be added by Supabase or manually]

-- Define an ENUM type for participant status
CREATE TYPE participant_status AS ENUM ('pending_payment', 'confirmed', 'cancelled_by_user', 'cancelled_ride');

-- Create the participants table
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),           -- Primary key, auto-generated UUID
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- Foreign key referencing the participating user
    ride_id UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE, -- Foreign key referencing the ride being joined
    status participant_status DEFAULT 'pending_payment' NOT NULL, -- Status of the participation (pending, confirmed, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,            -- Timestamp when the participation record was created
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,            -- Timestamp when the participation record was last updated

    -- Constraint to ensure a user can only join a specific ride once
    CONSTRAINT unique_user_ride UNIQUE (user_id, ride_id)
);

-- Add comments to the table and columns
COMMENT ON TABLE participants IS 'Links users to the rides they have joined or requested to join.';
COMMENT ON COLUMN participants.id IS 'Unique identifier for the participation record (UUID)';
COMMENT ON COLUMN participants.user_id IS 'Identifier of the user joining the ride (references users.id)';
COMMENT ON COLUMN participants.ride_id IS 'Identifier of the ride being joined (references rides.id)';
COMMENT ON COLUMN participants.status IS 'Current status of the participation (pending_payment, confirmed, cancelled_by_user, cancelled_ride)';
COMMENT ON COLUMN participants.created_at IS 'Timestamp of participation creation';
COMMENT ON COLUMN participants.updated_at IS 'Timestamp of last participation update';
COMMENT ON CONSTRAINT unique_user_ride ON participants IS 'Ensures that a user can join a specific ride only once.';


-- Create indexes for faster querying
CREATE INDEX idx_participants_user_id ON participants(user_id);
CREATE INDEX idx_participants_ride_id ON participants(ride_id);
CREATE INDEX idx_participants_status ON participants(status);

-- Trigger to automatically update updated_at on participants table update
-- (Uses the function created in the first migration)
CREATE TRIGGER update_participants_updated_at
BEFORE UPDATE ON participants
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();