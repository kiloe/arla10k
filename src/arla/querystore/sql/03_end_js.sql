
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


-- execute a mutation
CREATE OR REPLACE FUNCTION arla_exec(mutation json) RETURNS json AS $$
	return JSON.stringify(plv8.arla.exec(mutation));
$$ LANGUAGE "plv8";

-- execute a mutation using a json representation of the mutation
CREATE OR REPLACE FUNCTION arla_replay(mutation json) RETURNS boolean AS $$
	return plv8.arla.replay(mutation);
$$ LANGUAGE "plv8";

-- use graphql to execute a query
CREATE OR REPLACE FUNCTION arla_query(query json) RETURNS json AS $$
	return JSON.stringify(plv8.arla.query(query));
$$ LANGUAGE "plv8";

-- run the authentication func
CREATE OR REPLACE FUNCTION arla_authenticate(vals json) RETURNS json AS $$
	return JSON.stringify(plv8.arla.authenticate(vals));
$$ LANGUAGE "plv8";

-- run the registration transformation func
CREATE OR REPLACE FUNCTION arla_register(vals json) RETURNS json AS $$
	return JSON.stringify(plv8.arla.register(vals));
$$ LANGUAGE "plv8";

-- fetch the user config
CREATE OR REPLACE FUNCTION arla_info() RETURNS json AS $$
	return JSON.stringify({
		version: plv8.arla.cfg.version,
		mutations: Object.keys(plv8.arla.cfg.actions)
	});
$$ LANGUAGE "plv8";
