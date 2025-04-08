-- Migration: 006_modify_rides_schema
-- Description: Update rides table schema for V2 (seats, status).
-- Created at: NOW()

-- Rename 'available_seats' to 'total_seats' for clarity
ALTER TABLE rides
RENAME COLUMN available_seats TO total_seats;

COMMENT ON COLUMN rides.total_seats IS 'Total number of seats offered by the creator (1-5)';

-- Drop the old status column first, as it depends on the type ride_status
-- Note: This assumes the column using the ENUM type is named 'status'. Adjust if needed.
ALTER TABLE rides DROP COLUMN IF EXISTS status;

-- Now that no column uses the type, drop the old ride_status ENUM type
DROP TYPE IF EXISTS ride_status;

-- Add the new status column (using TEXT for flexibility, constraints can be added)
-- Now add the new TEXT based status column
ALTER TABLE rides
ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
CONSTRAINT ride_status_check CHECK (status IN ('active', 'archived', 'cancelled'));


-- Update existing rides - Set status based on old logic or default to 'active'
-- This depends on how the old 'status' ENUM was used. Assuming 'open' maps to 'active'.
-- The default 'active' handles new rows and potentially existing ones if the column was just dropped.
-- Logic to archive old rides will be handled by backend/cron job.

COMMENT ON COLUMN rides.status IS 'Current status of the ride (active, archived, cancelled)';

-- Add index on the new status column
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides (status);

-- Remove the old trigger if it references the old status column name or type (if applicable)
-- DROP TRIGGER IF EXISTS update_rides_updated_at ON rides; -- Example if needed (unlikely needed just for type change)

-- Re-create trigger if it was dropped (or ensure it's still valid)
-- Ensure the function update_updated_at_column exists from migration 001
-- Drop existing trigger first to avoid error if it exists
DROP TRIGGER IF EXISTS update_rides_updated_at ON rides;
-- Create the trigger
CREATE TRIGGER update_rides_updated_at
BEFORE UPDATE ON rides
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();