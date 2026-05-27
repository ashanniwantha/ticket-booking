-- Drop the trigger first (must be removed before dropping the table)
DROP TRIGGER IF EXISTS trg_users_set_updated_at ON users;

-- Drop the function used by the trigger
DROP FUNCTION IF EXISTS set_updated_at();

-- Finally drop the table
DROP TABLE IF EXISTS users;
