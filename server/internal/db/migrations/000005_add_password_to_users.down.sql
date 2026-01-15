-- Remove password_hash column from users table
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

