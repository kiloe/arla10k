--
-- NOTE: This file will be executed multiple times
---

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "plv8";

-- function executed for each new context to setup the environment
CREATE OR REPLACE FUNCTION public.plv8_init() RETURNS json AS $javascript$
	try {
		import * from "index.js"; // This is fake, but sematically correct
	} catch (e) {
		plv8.elog(ERROR, e.stack || e.message || e.toString());
	}
$javascript$ LANGUAGE "plv8";

-- fire normalizes trigger data into an "op" event that is emitted.
CREATE OR REPLACE FUNCTION arla_fire_trigger() RETURNS trigger AS $$
	return plv8.functions.arla_fire_trigger({
		opKind: TG_ARGV[0],
		op: TG_OP.toLowerCase(),
		table: TG_TABLE_NAME.toLowerCase(),
		record: TG_OP=="DELETE" ? OLD : NEW,
		old: TG_OP=="INSERT" ? {} : OLD,
		args: TG_ARGV.slice(1)
	});
$$ LANGUAGE "plv8";

-- migrate schema to match definitions
CREATE OR REPLACE FUNCTION arla_migrate() RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_migrate())
$$ LANGUAGE "plv8";

-- execute an action
CREATE OR REPLACE FUNCTION arla_exec(name text, args json) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_exec(name, args, false));
$$ LANGUAGE "plv8";
CREATE OR REPLACE FUNCTION arla_exec(name text, args json, replay boolean) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_exec(name, args, replay));
$$ LANGUAGE "plv8";

-- use graphql to execute a query
CREATE OR REPLACE FUNCTION arla_query(t text) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_query(t));
$$ LANGUAGE "plv8";

-- wipe all the data
CREATE OR REPLACE FUNCTION arla_destroy_data() RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_destroy_data());
$$ LANGUAGE "plv8";

-- since(x) === age(now(), x)
CREATE OR REPLACE FUNCTION since(t timestamptz) RETURNS interval AS $$
	select age(now(), t);
$$ LANGUAGE "sql" VOLATILE;

-- until(x) === age(x, now())
CREATE OR REPLACE FUNCTION until(t timestamptz) RETURNS interval AS $$
	select age(t, now());
$$ LANGUAGE "sql" VOLATILE;
