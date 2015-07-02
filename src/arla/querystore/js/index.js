import gql from "./graphql"

(function(){

	var listeners = {};
	var tests = [];
	var ddl = [];
	var schema = {};
	var actions = {};

	function action(name, fn){
		actions[name] = fn;
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

	function before(op, table, fn){
		addListener('before', op, table, fn);
	}

	function after(op, table, fn){
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

	function define(name, o){
		if( !o.properties ){
			o.properties = {};
		}
		if( !o.edges ){
			o.edges = {};
		}
		let alter = function(stmt){
			if(name != 'root'){
				ddl.push(stmt)
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

	function defineJoin(tables, o){
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
	var VIEWER_MATCH = /\$identity/g;

	function gqlToSql(viewer, {name, properties, edges}, {args, props, filters}, parent, i = 0){
		let cols = Object.keys(props || {}).map(function(k){
			var o = props[k];
			switch(o.kind){
			case 'property':
				if( !parent ){
					throw `the root entity does not have any properties: ${k} is not valid here`;
				}
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
				let jsonfn = edge.type == 'array' ? 'json_agg' : 'row_to_json';
				let type = (edge.type == 'array' ? edge.of : edge.type) || 'raw';
				if( type == 'raw' ){
					return `
						(with
							${q} as ( ${ sql } )
							select ${jsonfn}(${q}.*) from ${q}
						) as ${k}
					`;
				}
				let table = schema[type];
				if( !table ){
					throw `unknown return type ${type} for edge call ${k} on ${name}`
				}
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

	arla.trigger = function(e){
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
					console.log(`${ op } constraint rejected transaction for ${ e.table } record: ${ JSON.stringify(e.record, null, 4) }`);
					throw err;
				}
			});
		});
		return e.record;
	};

	arla.replay = function(mutation){
		arla.exec(mutation.ID, mutation.Name, mutation.Args, true);
		return true;
	};

	arla.exec = function(viewer, name, args, replay){
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
			console.debug(`action ${name} given`, args);
			var queryArgs = fn(...args);
			if( !queryArgs ){
				console.debug(`action ${name} was a noop`);
				return [];
			}
			console.debug(`action ${name} returned`, queryArgs);
			if( !Array.isArray(queryArgs) ){
				queryArgs = [queryArgs];
			}
			// ensure first arg is valid
			if( typeof queryArgs[0] != 'string' ){
				throw 'invalid response from action. should be: [sqlstring, ...args]';
			}
			// replace magic $viewer variable
			queryArgs[0] = queryArgs[0].replace(VIEWER_MATCH, plv8.quote_literal(viewer));
			// run
			return db.query(...queryArgs);
		}catch(err){
			if( !replay ){
				throw err;
			}
			if( !actions.resolver ){
				console.debug(`There is no 'resolver' function declared`);
				throw err;
			}
			var res = actions.resolver(err, name, args, actions);
			console.debug('action', name, args, 'initially failed, but was resolved')
			return res;
		}
	};

	arla.query = function(viewer, query){
		query = `root(){ ${query} }`;
		try{
			console.debug("QUERY:", viewer, query);
			let ast = gql.parse(query);
			//console.debug("AST:", ast);
			let sql = gqlToSql( viewer, schema.root, ast[0]);
			//console.debug(`SQL:`, sql);
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
	};

	arla.init = function(){
		try{
			// build schema
			ddl.forEach(function(stmt){
				db.query(stmt);
			});
			// Only options defined here are allowed
			let opts = {
				logLevel: console.INFO,
				actions: [],
			};
			for( let k in opts ){
				db.query(`
					insert into arla_config (key,value) values ($1::text, $2::json)
				`, k, JSON.stringify(opts[k]));
			}
			throw new Error('this is an error');
			// evaluate other config options
			for( let k in arla.cfg ){
				switch(k){
				case 'schema':
					break;
				default:
					if( opts[k] ){
						db.query(`
							update arla_config set value = $1 where key = $2
						`, arla.cfg[k], k);
					} else {
						throw 'no such config option: ' + k;
					}
				}
			}
		}catch(e){
			arla.throwError(e);
		}
	}

	arla.configure = function(cfg){
		if( arla.cfg ){
			throw 'configure should only be called ONCE!';
		}
		// setup user schema
		Object.keys(cfg.schema || {}).forEach(function(name){
			define(name, cfg.schema[name]);
		});
		// setup user actions
		let actionNames = Object.keys(cfg.actions || {})
		actionNames.forEach(function(name){
			action(name, cfg.actions[name]);
		});
		cfg.actions = actionNames;
		// setup the config table
		define('arla_config', {
			properties: {
				key:   {type: 'text', unique:true},
				value: {type: 'json'}
			},
			afterChange({key, value}){
				switch(key.toLowerCase()){
				case 'loglevel':
					console.logLevel = value;
					break;
				default:
					break;
				}
			}
		});
		// store cfg for later
		arla.cfg = cfg;
	}

	arla.throwError = function(e){
    plv8.elog(ERROR, e.stack || e.message || e.toString());
	}

})();

// Execute the user's code
try{
	//CONFIG//
} catch (e) {
		arla.throwError(e);
}
