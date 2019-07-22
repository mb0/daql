-- generated code

BEGIN;

CREATE SCHEMA mig;

CREATE TABLE mig.version (
	name text PRIMARY KEY,
	vers int8 NOT NULL,
	date timestamptz NULL,
	hash text NOT NULL
);

COMMIT;
