-- Migration: 005_modify_users_add_deleted_at
-- Description: Add soft delete capability to the users table.
-- Created at: NOW()

ALTER TABLE users
ADD COLUMN deleted_at TIMESTAMPTZ NULL; -- Add nullable timestamp for soft delete

COMMENT ON COLUMN users.deleted_at IS 'Timestamp when the user account was soft-deleted. NULL means active.';

-- Optional: Index for potentially filtering out deleted users frequently
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);