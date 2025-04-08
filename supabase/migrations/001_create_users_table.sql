-- Migration: 001_create_users_table
-- Description: Creates the initial users table to store user information.
-- Created at: NOW() [Timestamp will be added by Supabase or manually]

-- Enable UUID generation if not already enabled
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp"; -- Usually enabled by default on Supabase

-- Create the users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Primary key, auto-generated UUID
    email TEXT UNIQUE NOT NULL,                   -- User's email, must be unique and not empty
    password_hash TEXT NOT NULL,                  -- Hashed password, must not be empty
    first_name TEXT,                              -- User's first name (optional)
    last_name TEXT,                               -- User's last name (optional)
    birth_date DATE,                              -- User's date of birth (optional)
    nationality TEXT,                             -- User's nationality (optional)
    whatsapp TEXT UNIQUE NOT NULL,                -- User's WhatsApp number, must be unique and not empty for contact
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL, -- Timestamp when the user was created
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL  -- Timestamp when the user was last updated
);

-- Add comments to the table and columns for clarity
COMMENT ON TABLE users IS 'Stores user profile information for authentication and contact.';
COMMENT ON COLUMN users.id IS 'Unique identifier for the user (UUID)';
COMMENT ON COLUMN users.email IS 'User login email address (unique)';
COMMENT ON COLUMN users.password_hash IS 'Hashed user password';
COMMENT ON COLUMN users.first_name IS 'User''s first name';
COMMENT ON COLUMN users.last_name IS 'User''s last name';
COMMENT ON COLUMN users.birth_date IS 'User''s date of birth';
COMMENT ON COLUMN users.nationality IS 'User''s nationality';
COMMENT ON COLUMN users.whatsapp IS 'User''s WhatsApp number (unique, used for ride sharing contact)';
COMMENT ON COLUMN users.created_at IS 'Timestamp of user creation';
COMMENT ON COLUMN users.updated_at IS 'Timestamp of last user update';

-- Optional: Create an index on the email column for faster lookups
CREATE INDEX idx_users_email ON users(email);

-- Optional: Create an index on the whatsapp column for faster lookups
CREATE INDEX idx_users_whatsapp ON users(whatsapp);

-- Function to automatically update updated_at timestamp on row modification
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at on users table update
CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();