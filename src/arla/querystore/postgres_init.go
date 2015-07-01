package querystore

const postgresInitScript = `
--
-- NOTE: This file will be executed multiple times
---

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "plv8";

-- function executed for each new context to setup the environment
CREATE OR REPLACE FUNCTION public.plv8_init() RETURNS json AS $javascript$

	// polyfil
	if (!Object.assign) {
		Object.defineProperty(Object, 'assign', {
			enumerable: false,
			configurable: true,
			writable: true,
			value: function(target, firstSource) {
				'use strict';
				if (target === undefined || target === null) {
					throw new TypeError('Cannot convert first argument to object');
				}

				var to = Object(target);
				for (var i = 1; i < arguments.length; i++) {
					var nextSource = arguments[i];
					if (nextSource === undefined || nextSource === null) {
						continue;
					}

					var keysArray = Object.keys(Object(nextSource));
					for (var nextIndex = 0, len = keysArray.length; nextIndex < len; nextIndex++) {
						var nextKey = keysArray[nextIndex];
						var desc = Object.getOwnPropertyDescriptor(nextSource, nextKey);
						if (desc !== undefined && desc.enumerable) {
							to[nextKey] = nextSource[nextKey];
						}
					}
				}
				return to;
			}
		});
	}

	// arla global
	var arla = {};

	// add console logging
	var console = (function(console){

		console.NONE = 0;
		console.DEBUG = 1;
		console.INFO = 2;
		console.LOG = 3;
		console.WARN = 4;
		console.ERROR = 5;

		console.logLevel = o.ERROR;

		function logger(level, pglevel, ...msgs) {
			var msg = msgs.map(function(msg){
				if( typeof msg == 'object' ){
					msg = JSON.stringify(msg, null, 4);
				}
				return msg;
			}).join(' ');
			(msg || '').split(/\n/g).forEach(function(line){
				if( o.logLevel <= level   ){
					plv8.elog(level, line);
				}
			})
		}

		console.debug = function(...msg) {
			logger(o.DEBUG, NOTICE, ...msg)
		}

		console.info = function(...msg) {
			logger(o.INFO, INFO, ...msg)
		}

		console.log = function(...msg) {
			logger(o.LOG, NOTICE, ...msg)
		}

		console.warn = function(...msg) {
			logger(o.WARN, WARNING, ...msg)
		}

		console.error = function(...msg) {
			logger(o.ERROR, WARNING, ...msg)
		}

		return console;
	})({});

	// expose database
	var db = (function(db){

		db.query = function(sql, ...args){
			console.debug(sql, args);
			return plv8.execute(sql, ...args);
		}

		db.transaction = function(){
			plv8.subtransaction(function(){
				fn();
			});
		}

		return db;
	})({});

	//RUNTIME//

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
CREATE OR REPLACE FUNCTION arla_exec(viewer uuid, name text, args json) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_exec(viewer, name, args, false));
$$ LANGUAGE "plv8";

CREATE OR REPLACE FUNCTION arla_exec(viewer uuid, name text, args json, replay boolean) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_exec(viewer, name, args, replay));
$$ LANGUAGE "plv8";

CREATE OR REPLACE FUNCTION arla_replay(mutation json) RETURNS boolean AS $$
	return plv8.functions.arla_replay(mutation);
$$ LANGUAGE "plv8";

-- use graphql to execute a query
CREATE OR REPLACE FUNCTION arla_query(viewer uuid, t text) RETURNS json AS $$
	return JSON.stringify(plv8.functions.arla_query(viewer, t));
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

SELECT arla_migrate();
`
