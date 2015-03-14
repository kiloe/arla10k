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
	if( !o.edges ){
		o.edges = {};
	}
	let alter = function(s){
		if(name != 'root'){
			ddl.push(s)
		}
	}
	o.name = name;
	alter(`CREATE TABLE ${ plv8.quote_ident(name) } ()`);
	alter(`CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('before')`);
	alter(`CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('after')`);
	for(let k in o.properties){
		alter(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN ${ plv8.quote_ident(k) } ${ col(o.properties[k]) }`);
	}
	if( !o.properties.id ){
		alter(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN id UUID PRIMARY KEY DEFAULT uuid_generate_v4()`);
		o.properties.id = {type:'uuid'};
	}
	if( o.indexes ){
		for(let k in o.indexes){
			let idx = o.indexes[k];
			let using = idx.using ? `USING ${ idx.using }` : '';
			alter(`CREATE ${ idx.unique ? 'UNIQUE' : '' } INDEX ${ name }_${ k }_idx ON ${ plv8.quote_ident(name) } ${ using } ( ${ idx.on.map(c => plv8.quote_ident(c) ).join(',') } )` );
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

var PARENT_MATCH = /\$this/g;
var VIEWER_MATCH = /\$viewer/g;

function gqlToSql(viewer, {name, properties, edges}, {args, props, filters}, parent, i = 0){
	let cols = Object.keys(props || {}).map(function(k){
		var o = props[k];
		switch(o.kind){
		case 'property':
			return `${ parent }.${ k }`;
		case 'edge':
			let x = `x${ i }`;
			let q = `q${ i }`;
			let call = edges[k];
			if( !call ){
				throw `no such edge ${k} for ${name}`;
			}
			let edge = call(...o.args);
			if( !edge.query ){
				throw `missing query for edge call ${k} on ${name}`;
			}
			let sql = edge.query;
			// Replace special variables
			sql = sql.replace(VIEWER_MATCH, plv8.quote_literal(viewer));
			sql = sql.replace(PARENT_MATCH, function(match){
				if( !parent ){
					throw "Cannot use $this table replacement on root calls";
				}
				return plv8.quote_ident(parent);
			});
			let type = edge.type == 'array' ? edge.of : edge.type;
			let table = schema[type];
			if( !table ){
				throw `unknown return type ${type} for edge call ${k} on ${name}`
			}
			let jsonfn = edge.type == 'array' ? 'json_agg' : 'row_to_json';
			return `
				(with
					${q} as ( ${ sql } ),
					${x} as ( ${gqlToSql(viewer, table, o, q, ++i)} from ${q} )
					select ${jsonfn}(${x}.*) from ${x}
				) as ${k}
			`;
		default:
			throw `unknown property type: ${o.kind}`
		}
	});
	return `select ${cols.join(',')}`;
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
			if( name == 'root'){
				continue;
			}
			db.query(`delete from ${plv8.quote_ident(name)}`)
		}
		return true;
	},
	arla_exec(name, args, replay){
		if( name == 'resolver' ){
			throw "no such action resolver"; // HACK
		}
		var fn = actions[name];
		if( !fn ){
			if( /^[a-zA-Z0-9_]+$/.test(name) ){
				throw `no such action ${name}`;
			} else {
				throw 'invalid action';
			}
		}
		try{
			return fn(...args);
		}catch(err){
			if( !replay ){
				throw err;
			}
			if( !actions.resolver ){
				console.log(`There is no 'resolver' function declared`);
				throw err;
			}
			var res = actions.resolver.bind(db)(err, name, args, actions);
			console.debug('action', name, args, 'initially failed, but was resolved')
			return res;
		}
	},
	arla_query(viewer, query){
		query = `root(){ ${query} }`;
		try{
			console.debug("QUERY:", viewer, query);
			let ast = gql.parse(query);
			console.debug("AST:", ast);
			let sql = gqlToSql( viewer, schema.root, ast[0]);
			console.debug(`SQL:`, sql);
			let res = db.query(sql)[0];
			console.debug("RESULT", res);
			return res;
		}catch(err){
			if( err.line && err.offset ){
				console.warn( query.split(/\n/)[err.line-1] );
				console.warn( `${ Array(err.column).join('-') }^` );
				throw new SyntaxError(`arla_query: line ${err.line}, column ${err.column}: ${err.message}`)
			}
			throw err;
		}
	}
}
