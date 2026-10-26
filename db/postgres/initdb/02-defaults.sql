-- Cluster defaults: every session gets UTC and UTF-8.
--
-- ALTER DATABASE/ROLE need a literal name, so the names are resolved at
-- runtime through a DO block instead of relying on shell environment
-- variables.
DO $defaults$
DECLARE
    db_name   text := current_database();
    role_name text := current_user;
BEGIN
    EXECUTE format('ALTER DATABASE %I SET timezone TO %L', db_name, 'UTC');
    EXECUTE format('ALTER DATABASE %I SET client_encoding TO %L', db_name, 'UTF8');
    EXECUTE format('ALTER ROLE %I SET timezone TO %L', role_name, 'UTC');

    -- Future databases inherit the same defaults.
    EXECUTE format('ALTER DATABASE %I SET timezone TO %L', 'template1', 'UTC');
    EXECUTE format('ALTER DATABASE %I SET client_encoding TO %L', 'template1', 'UTF8');
END
$defaults$;
