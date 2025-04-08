-- Migration: 007_modify_participants_status
-- Description: Update participants table status column for V2.
-- Created at: NOW()

-- Drop the old status column first, as it depends on the type participant_status
ALTER TABLE participants DROP COLUMN IF EXISTS status;

-- Now that no column uses the type, drop the old participant_status ENUM type
DROP TYPE IF EXISTS participant_status;

-- Add the new status column (using TEXT for flexibility)
-- Now add the new TEXT based status column
ALTER TABLE participants
ADD COLUMN status TEXT NOT NULL DEFAULT 'pending_payment'
CONSTRAINT participant_status_check CHECK (status IN ('pending_payment', 'active', 'left', 'cancelled_ride'));

-- Update existing participants - Map old statuses to new ones if necessary.
-- Example: Map 'confirmed' to 'active'. 'pending_payment' remains. Others might map to 'left' or 'cancelled_ride'.
-- UPDATE participants SET status = 'active' WHERE status = 'confirmed'; -- Example, adjust based on old values
-- UPDATE participants SET status = 'left' WHERE status = 'cancelled_by_user'; -- Example
-- UPDATE participants SET status = 'cancelled_ride' WHERE status = 'cancelled_ride'; -- Example (if name was same)
-- Since we dropped the column, existing rows will get the new default 'pending_payment'.
-- Manual update might be needed based on previous data state.

COMMENT ON COLUMN participants.status IS 'Current status of the participation (pending_payment, active, left, cancelled_ride)';

-- Re-create or ensure index exists on the new status column
DROP INDEX IF EXISTS idx_participants_status;
CREATE INDEX idx_participants_status ON participants (status);

-- Re-create or ensure trigger exists
DROP TRIGGER IF EXISTS update_participants_updated_at ON participants;
CREATE TRIGGER update_participants_updated_at
BEFORE UPDATE ON participants
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();