-- generated code

BEGIN;

CREATE SCHEMA evt;

CREATE TABLE evt.audit (
	rev timestamptz PRIMARY KEY,
	created timestamptz NOT NULL,
	arrived timestamptz NOT NULL,
	acct uuid NULL,
	extra jsonb NULL
);

CREATE TABLE evt.event (
	id serial8 PRIMARY KEY,
	rev timestamptz NOT NULL,
	top text NOT NULL,
	key text NOT NULL,
	cmd text NOT NULL,
	arg jsonb NULL
);

COMMIT;
