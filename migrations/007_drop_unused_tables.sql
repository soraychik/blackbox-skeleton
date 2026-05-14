-- Drop legacy tables that are no longer referenced in application code.
-- Order matters: audit has a FK on users, so it must be dropped first.

-- Replaced by audit_log (migration 005)
DROP TABLE IF EXISTS audit;

-- Legacy local auth; authentication is now handled via LDAP (system_settings)
DROP TABLE IF EXISTS users;

-- Background job infrastructure that was never wired into application logic
DROP TABLE IF EXISTS jobs;
