
$javascript$ LANGUAGE "plv8";

-- fire normalizes trigger data into an "op" event that is emitted.
CREATE OR REPLACE FUNCTION arla_fire_trigger() RETURNS trigger AS $$
	return plv8.arla.trigger({
		opKind: TG_ARGV[0],
		op: TG_OP.toLowerCase(),
		table: TG_TABLE_NAME.toLowerCase(),
		record: TG_OP=="DELETE" ? OLD : NEW,
		old: TG_OP=="INSERT" ? {} : OLD,
		args: TG_ARGV.slice(1)
	});
$$ LANGUAGE "plv8";


-- execute an action
CREATE OR REPLACE FUNCTION arla_exec(viewer uuid, name text, args json) RETURNS json AS $$
	return JSON.stringify(plv8.arla.exec(viewer, name, args, false));
$$ LANGUAGE "plv8";

CREATE OR REPLACE FUNCTION arla_exec(viewer uuid, name text, args json, replay boolean) RETURNS json AS $$
	return JSON.stringify(plv8.arla.exec(viewer, name, args, replay));
$$ LANGUAGE "plv8";

CREATE OR REPLACE FUNCTION arla_replay(mutation json) RETURNS boolean AS $$
	return plv8.arla.replay(mutation);
$$ LANGUAGE "plv8";

-- use graphql to execute a query
CREATE OR REPLACE FUNCTION arla_query(viewer uuid, t text) RETURNS json AS $$
	return JSON.stringify(plv8.arla.query(viewer, t));
$$ LANGUAGE "plv8";
