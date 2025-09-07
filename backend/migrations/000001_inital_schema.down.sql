-- Drop triggers and functions
DROP TRIGGER IF EXISTS update_tasks_modtime ON tasks;

DROP TRIGGER IF EXISTS update_users_modtime ON users;

DROP FUNCTION IF EXISTS update_timestamp_column ();

-- Drop tables in reverse order of creation due to foreign key constraints
DROP TABLE IF EXISTS task_executions;

DROP TABLE IF EXISTS tasks;

DROP TABLE IF EXISTS workers;

DROP TABLE IF EXISTS user_roles;

DROP TABLE IF EXISTS roles;

DROP TABLE IF EXISTS users;

-- Drop custom types
DROP TYPE IF EXISTS task_status;

-- Drop extensions
DROP EXTENSION IF EXISTS "pgcrypto";