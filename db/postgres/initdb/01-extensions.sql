-- Extensions enabled by default in every database created by this image.
--
--   vector     : pgvector 0.8.7 (embedded vector similarity search)
--   uuid-ossp  : UUID generation and comparison operators
--   pgcrypto   : cryptographic functions and gen_random_uuid()
--
-- Note: since PostgreSQL 13 the core server also provides gen_random_uuid(),
-- so UUIDs work even without pgcrypto. Both are enabled to cover every
-- client expectation.
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
