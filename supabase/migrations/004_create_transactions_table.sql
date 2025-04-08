-- Migration: 004_create_transactions_table
-- Description: Creates the transactions table to store payment information related to rides.
-- Created at: NOW() [Timestamp will be added by Supabase or manually]

-- Define an ENUM type for transaction status
CREATE TYPE transaction_status AS ENUM ('pending', 'succeeded', 'failed', 'refunded'); -- Added refunded for potential future use

-- Create the transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),           -- Primary key, auto-generated UUID
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL, -- Foreign key referencing the user who paid (SET NULL if user deleted)
    ride_id UUID NOT NULL REFERENCES rides(id) ON DELETE CASCADE,  -- Foreign key referencing the associated ride
    participant_id UUID UNIQUE REFERENCES participants(id) ON DELETE SET NULL, -- Optional: Link directly to the participation record (SET NULL if participant deleted)
    stripe_payment_intent_id TEXT UNIQUE NOT NULL,           -- Unique ID from Stripe Payment Intent
    status transaction_status DEFAULT 'pending' NOT NULL,   -- Status of the transaction (pending, succeeded, failed)
    amount INTEGER NOT NULL,                                 -- Amount paid (in cents, e.g., 200 for 2 EUR)
    currency CHAR(3) NOT NULL,                               -- Currency code (e.g., 'EUR')
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,            -- Timestamp when the transaction record was created
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL             -- Timestamp when the transaction record was last updated
);

-- Add comments to the table and columns
COMMENT ON TABLE transactions IS 'Stores details about payment transactions processed via Stripe for ride participation.';
COMMENT ON COLUMN transactions.id IS 'Unique identifier for the transaction record (UUID)';
COMMENT ON COLUMN transactions.user_id IS 'Identifier of the user who initiated the payment (references users.id)';
COMMENT ON COLUMN transactions.ride_id IS 'Identifier of the ride associated with the payment (references rides.id)';
COMMENT ON COLUMN transactions.participant_id IS 'Identifier of the specific participation record this transaction confirms (references participants.id)';
COMMENT ON COLUMN transactions.stripe_payment_intent_id IS 'Unique identifier for the payment intent provided by Stripe';
COMMENT ON COLUMN transactions.status IS 'Current status of the transaction (pending, succeeded, failed, refunded)';
COMMENT ON COLUMN transactions.amount IS 'Transaction amount in the smallest currency unit (e.g., cents)';
COMMENT ON COLUMN transactions.currency IS 'ISO currency code (e.g., EUR, USD)';
COMMENT ON COLUMN transactions.created_at IS 'Timestamp of transaction creation';
COMMENT ON COLUMN transactions.updated_at IS 'Timestamp of last transaction update';

-- Create indexes for faster querying
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_ride_id ON transactions(ride_id);
CREATE INDEX idx_transactions_participant_id ON transactions(participant_id);
CREATE INDEX idx_transactions_stripe_payment_intent_id ON transactions(stripe_payment_intent_id);
CREATE INDEX idx_transactions_status ON transactions(status);

-- Trigger to automatically update updated_at on transactions table update
-- (Uses the function created in the first migration)
CREATE TRIGGER update_transactions_updated_at
BEFORE UPDATE ON transactions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();