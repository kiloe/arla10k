import "./polyfill";
import * as assert from "./assert";
import console from "./console";
import db from "./db";
import gql from "./graphql"

var SHOW_DEBUG = true;

var listeners = {};
var tests = [];
var schema = {};
var ddl = [];
var actions = {};

export function action(name, fn){
	actions[name] = fn.bind(db);
}

function addListener(kind, op, table, fn){
	table.trim().split(/\s/g).forEach(function(table){
		listeners[table] = op.trim().split(/\s/g).reduce(function(ops, op){
			ops[kind + '-' + op].push(fn);
			return ops;
		}, listeners[table] || {
			'before-update': [],
			'after-update' : [],
			'before-insert': [],
			'after-insert' : [],
			'before-delete': [],
			'after-delete' : []
		});
	});
}

export function before(op, table, fn){
	addListener('before', op, table, fn);
}

export function after(op, table, fn){
	addListener('after', op, table, fn);
}

function col({type = 'text', nullable = false, def = undefined, onDelete = 'CASCADE', onUpdate = 'RESTRICT', ref} = {}) {
	if( type == 'timestamp' ){
		type = 'timestamptz'; // never use non timezone stamp - it's bad news.
	}
	var x = [type];
	if( ref ){
		x.push(`REFERENCES ${ plv8.quote_ident(ref) }`);
		x.push(`ON DELETE ${ onDelete }`);
		x.push(`ON UPDATE ${ onUpdate }`);
	}
	if( !nullable ){
		x.push('NOT NULL');
	}
	if( def === undefined ){
		switch(type){
		case 'boolean':   def = "false";   break;
		case 'json':      def = "'{}'";    break;
		case 'timestampz':def = 'now()';   break;
		default:          def = null;      break;
		}
	}
	if( def !== null ){
		x.push(`DEFAULT ${ def }`);
	}
	return x.join(' ');
}

export function define(name, o){
	if( !o.properties ){
		o.properties = {};
	}
	if( !o.refs ){
		o.refs = {};
	}
	o.name = name;
	ddl.push(`CREATE TABLE ${ plv8.quote_ident(name) } ()`);
	ddl.push(`CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('before')`);
	ddl.push(`CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('after')`);
	for(let k in o.properties){
		ddl.push(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN ${ plv8.quote_ident(k) } ${ col(o.properties[k]) }`);
	}
	if( !o.properties.id ){
		ddl.push(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN id UUID PRIMARY KEY DEFAULT uuid_generate_v4()`);
		o.properties.id = {type:'uuid'};
	}
	for(let k in o.refs){
		let ref = o.refs[k];
		if( !ref.hasOne ){
			continue;
		}
		let key = ref.hasOne + '_id';
		ddl.push(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN ${ plv8.quote_ident(key) } ${ col({type: 'uuid', ref: ref.hasOne}) }`);
		ddl.push(`CREATE INDEX ${ name }_${ key }_idx ON ${ plv8.quote_ident(name) } ( ${ plv8.quote_ident(key) } )`);
	}
	if( o.indexes ){
		for(let k in o.indexes){
			let idx = o.indexes[k];
			let using = idx.using ? `USING ${ idx.using }` : '';
			ddl.push(`CREATE ${ idx.unique ? 'UNIQUE' : '' } INDEX ${ name }_${ k }_idx ON ${ plv8.quote_ident(name) } ${ using } ( ${ idx.on.map(c => plv8.quote_ident(c) ).join(',') } )` );
		}
	}
	if( o.beforeChange ){
		before('insert update', name, o.beforeChange);
	}
	if( o.afterChange ){
		after('insert update', name, o.afterChange);
	}
	if( o.beforeUpdate ){
		before('update', name, o.beforeUpdate);
	}
	if( o.afterUpdate ){
		after('update', name, o.afterUpdate);
	}
	if( o.beforeInsert ){
		before('insert', name, o.beforeInsert);
	}
	if( o.afterInsert ){
		after('insert', name, o.afterInsert);
	}
	if( o.beforeDelete ){
		before('delete', name, o.beforeDelete);
	}
	if( o.afterDelete ){
		after('delete', name, o.afterDelete);
	}
	schema[name] = o;
}

export function defineJoin(tables, o){
	if( !o ){
		o = {};
	}
	o.properties = Object.assign(o.properties || {}, tables.reduce(function(o, name){
		o[ `${ name }_id` ] = {type: 'uuid', ref:name};
		return o;
	}, {}));
	o.indexes = Object.assign(o.indexes || {}, {
		join: {
			unique: true,
		on: tables.map(t => `${t}_id`)
		}
	})
	define(tables.join('_'), o);
}


function gqlToSql({name, properties, refs}, {id, props, filters}, p, p_key, r = null, i = 0){
	var t = `t${ i }`;
	var v = `v${ i }`;
	var cols = Object.keys(props || {}).map(function(k){
		var o = props[k];
		if( o.kind == 'property' ){
			if( properties[k] ){
				return `${ t }.${ k }`;
			} else if( r && r.via && schema[r.via].properties[k] ){
				return `${ v }.${ k }`;
			}
			throw "no such column "+k+" for "+name;
		}
		var r2 = refs[k];
		if( !r2 ){
			throw "no such edge "+k+" for "+name;
		}
		var x = `x${ i }`;
		return `(with ${x} as ( ${ gqlToSql(schema[r2.hasOne || r2.hasMany], o, t, `${ name }_id`, r2, ++i) }) select json_agg(${x}.*) from ${x} ) as ${ k }`;
	});
	var filters = Object.keys(filters || {}).map(function(k){
		return '';
	})
	if( r ){
		if( r.hasMany ){
			if( r.via ){
				return `select ${ cols.join(',') } from ${ name } ${ t } left join ${ r.via } ${ v } on ${ t }.id = ${ v }.${ name }_id where ${ v }.${ p_key } = ${ p }.id`;
			} else {
				return `select ${ cols.join(',') } from ${ name } ${ t } where ${ t }.${ p_key } = ${ p }.id`;
			}
		} else if( r.hasOne ){
			return `select ${ cols.join(',') } from ${ name } ${ t } where ${ t }.id = ${ p }.${ name }_id`;
		}
	}
	return `select ${ cols.join(',') } from ${ name } ${ t } where ${ t }.id = ${ plv8.quote_literal(id) }`;
}

define('meta', {
	properties: {
		key:   {type: 'text', unique: true},
		value: {type:'text'}
	}
});

// Functions that will be exposed via SQL
export var sql_exports = {
	arla_fire_trigger(e){
		let op = e.opKind + '-' + e.op;
		['*', e.table].forEach(function(table){
			let ops = listeners[table]
			if( !ops || ops.length == 0){
				return;
			}
			let triggers = ops[op];
			if( !triggers ){
				return;
			}
			triggers.forEach(function(fn){
				try{
					fn(e.record, e);
				}catch(err){
					console.debug(`${ op } constraint rejected transaction for ${ e.table } record: ${ JSON.stringify(e.record, null, 4) }`);
					throw err;
				}
			});
		});
		return e.record;
	},
	arla_migrate(){
		db.transaction(function(){
			ddl.forEach(function(stmt){
				try{
					db.query(stmt);
				}catch(err){
					if( (/already exists/i).test( err,toString() ) ){
						// ignore ALTER TABLE errors when column exists
					} else {
						throw err;
					}
				}
			})
		})
	},
	arla_destroy_data(){
		for(let name in schema){
			db.query(`delete from ${plv8.quote_ident(name)}`)
		}
		return true;
	},
	arla_exec(name, args){
		var fn = actions[name];
		if( !fn ){
			if( /^[a-zA-Z0-9_]+$/.test(name) ){
				throw `no such action ${name}`;
			} else {
				throw 'invalid action';
			}
		}
		return fn(...args);
	},
	arla_query(query){
		try{
			var ast = gql.parse(query);
			var table = schema[ ast.name ];
			if( !table ){
				throw "no relation found named "+ast.name;
			}
			console.debug("AST", ast);
			var sql = gqlToSql( table, ast );
			console.debug("SQL", sql);
			var res = {};
			res[ast.name] = db.query(sql)[0];
			console.debug("RESULT", res);
			return res;
		}catch(err){
			if( err.line && err.offset ){
				console.warn( query.split(/\n/)[err.line-1] );
				console.warn( `${ Array(err.column).join('-') }^` );
				throw new SyntaxError(`arla_query: line ${err.line}, column ${err.column}: ${err.message}`)
			}
			console.warn(ast);
			console.warn(sql);
			throw err;
		}
	}
}
