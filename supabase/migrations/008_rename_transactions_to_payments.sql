-- Migration: 008_rename_transactions_to_payments
-- Description: Rename transactions table and update its status column.
-- Created at: NOW()

-- Rename the table
ALTER TABLE transactions RENAME TO payments;

-- Drop the old status column first, as it depends on the type transaction_status
-- Note: We use the new table name 'payments' here because the rename happened earlier in the script.
ALTER TABLE payments DROP COLUMN IF EXISTS status;

-- Now that no column uses the type, drop the old transaction_status ENUM type
DROP TYPE IF EXISTS transaction_status;

-- Add the new status column (using TEXT)
-- Now add the new TEXT based status column
ALTER TABLE payments
ADD COLUMN status TEXT NOT NULL DEFAULT 'pending' -- Keep pending default temporarily, backend should update to succeeded/failed
CONSTRAINT payment_status_check CHECK (status IN ('pending', 'succeeded', 'failed')); -- Allow pending initially

-- Update existing payments - Map old statuses to new ones.
-- Example: Map 'succeeded' or 'paid' to 'succeeded'. Map 'failed' to 'failed'.
-- UPDATE payments SET status = 'succeeded' WHERE status = 'paid'; -- Example
-- Since we dropped the column, manual update or backend logic on first read might be needed.

COMMENT ON TABLE payments IS 'Stores details about payment transactions processed via Stripe.';
COMMENT ON COLUMN payments.status IS 'Status of the payment (pending, succeeded, failed)';

-- Rename indexes (optional but good practice) - Adjust names as needed
-- Drop old indexes first (using old table name in index name convention)
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_transactions_ride_id;
DROP INDEX IF EXISTS idx_transactions_participant_id;
DROP INDEX IF EXISTS idx_transactions_stripe_payment_intent_id;
DROP INDEX IF EXISTS idx_transactions_status;
-- Create new indexes with new table name convention
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_ride_id ON payments(ride_id);
CREATE INDEX IF NOT EXISTS idx_payments_participant_id ON payments(participant_id);
CREATE INDEX IF NOT EXISTS idx_payments_stripe_payment_intent_id ON payments(stripe_payment_intent_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);


-- Re-create or ensure trigger exists
DROP TRIGGER IF EXISTS update_transactions_updated_at ON payments; -- Use old name if trigger wasn't renamed automatically
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
CREATE TRIGGER update_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();