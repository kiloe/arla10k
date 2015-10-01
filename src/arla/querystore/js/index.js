import gql from './graphql';

// UserError should be used when you want an error message to
// be shown to an end user.
class UserError extends Error {
	constructor(m) {
		var err = super(m);
		Object.assign(this, {
			name: 'UserError',
			message: m,
			stack: err.stack
		});
	}
}

// QueryError is returned when query parsing fails.
// It contains the line and context of the error
class QueryError extends Error {
	constructor(o) {
		var err = super(o.message);
		o.error = o.message;
		delete o.message;
		Object.assign(this, {
			name: 'QueryError',
			message: JSON.stringify(o),
			stack: err.stack
		});
	}
}

// MutationError is returned when exec fails.
// It contains the entire mutation
class MutationError extends Error {
	constructor(o) {
		var err = super(o.message);
		o.error = o.message;
		delete o.message;
		Object.assign(this, {
			name: 'MutationError',
			message: JSON.stringify(o),
			stack: err.stack,
		});
	}
}

(function(){

	var listeners = {};
	var ddl = [];
	var schema = {};
	var actions = {};

	const ARRAY_OF_SIMPLE = 1;
	const ARRAY_OF_OBJECTS = 2;
	const SIMPLE = 3;
	const OBJECT = 4;

	function action(name, fn){
		actions[name] = fn;
	}

	function addListener(kind, op, klass, fn){
		fn.klass = klass;
		listeners[klass.name] = op.trim().split(/\s/g).reduce(function(ops, op){
			ops[kind + '-' + op].push(fn);
			return ops;
		}, listeners[klass.name] || {
			'before-update': [],
			'after-update' : [],
			'before-insert': [],
			'after-insert' : [],
			'before-delete': [],
			'after-delete' : []
		});
	}

	// Convert type klass to pg type string
	function typeToString(t){
		if( typeof t == 'undefined' ){
			throw new UserError(`invalid type 'undefined'`);
		}
		else if( typeof t == 'string' ){
			t = t;
		}
		else if( t === String ){
			t = 'text';
		}
		else if( t === Number ){
			t = 'float';
		}
		else if( t === Date ){
			t = 'timestamptz';
		}
		else if( t === Array ){
			t = 'array';
		}
		else if( t === Boolean ){
			t = 'boolean';
		}
		else if ( t === Object ){
			t = 'jsonb';
		}
		else if( typeof t == 'function' && t.name ){
			if( !schema[t.name] ){
				define(t.name, t);
			}
			t = t.name;
		}
		else {
			throw new UserError(`invalid type for property: ${t}`);
		}
		return t;
	}

	// build column definition
	function col({type = 'text', nullable = false, def = undefined, pk = false, onDelete = 'CASCADE', onUpdate = 'RESTRICT', ref} = {}) {
		let t = type;
		if( t == 'timestamp' ){
			console.warn('there are issues with the timestamp type it is recordmend you use timestamptz');
		}
		if( !t && ref ){
			t = 'uuid';
		}
		if( t == 'array' ){
			t = 'jsonb';
		}
		var x = [t];
		if( ref ){
			x.push(`REFERENCES ${ plv8.quote_ident(ref) }`);
			x.push(`ON DELETE ${ onDelete }`);
			x.push(`ON UPDATE ${ onUpdate }`);
		}
		if( !nullable ){
			x.push('NOT NULL');
		}
		if( def === undefined && !nullable ){
			switch(t){
				case 'text':      def = `''`;                            break;
				case 'integer':   def = `0`;                             break;
				case 'boolean':   def = `false`;                         break;
				case 'json':
				case 'jsonb':     def = type=='array' ? `'[]'` : `'{}'`; break;
				case 'timestampz':def = `now()`;                         break;
			}
		}
		if( pk ){
			x.push('PRIMARY KEY');
		}
		if( def !== null && def !== undefined ){
			x.push(`DEFAULT ${ def }`);
		}
		return x.join(' ');
	}

	function define(name, klass){
		if( !klass ){
			throw new Error(`invalid class for ${name} ${typeof klass}`);
		}
		if( schema[name] ){
			if( schema[name] != klass ){
				console.warn(`entity type ${name} is already defined`);
			}
			return;
		}
		schema[name] = klass;
		if( klass.requires ){
			for(let req in klass.requires){
				define(req, klass.requires[req]);
			}
		}
		if( !klass.props ){
			throw UserError('class '+klass.name+' has no props');
		}
		let columns = Object.keys(klass.props).reduce(function(props, k){
			let prop = klass.props[k];
			prop.name = k;
			if(prop.hasOwnProperty('ref')){
				if( !prop.ref ){
					throw new UserError(`invalid ref for property ${prop.name}: cannot be ${typeof prop.ref}`);
				}
				prop.ref = typeToString(prop.ref);
				prop.type = 'uuid';
			}
			if(!prop.type){
				prop.type = 'text';
			}
			if( prop.of ){
				prop.of = typeToString(prop.of);
				if( prop.of == 'array' ){
					throw new UserError(`'of' cannot be 'array'`);
				}
			}
			prop.type = typeToString(prop.type, prop.of);
			prop.klass = klass;
			if(prop.query){
				// add the magic _count property
				if( prop.type == 'array' && !klass.props[prop.name+'_count']){
					klass[prop.name+'_count'] = {
						name: prop.name+'_count',
						klass: klass,
						type: 'counter',
						queryName: prop.name
					};
				}
				return props;
			}
			props.push(prop);
			return props;
		},[]);

		// Add a DDL stmt to be executed during startup.
		// Priority sets the order of the statement.
		let alter = function(stmt, priority){
			if(name != 'root'){
				ddl.push({
					priority:priority || 0,
					stmt: stmt,
				});
			}
		};
		alter(`CREATE TABLE ${ plv8.quote_ident(name) } ()`);
		alter(`CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('before')`, 10);
		alter(`CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('after')`, 10);
		columns.forEach(function(p){
			let priority = p.pk ? 1 : 2;
			alter(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN ${ plv8.quote_ident(p.name) } ${ col(p) }`, priority);
			if(p.unique){
				alter(`CREATE UNIQUE INDEX ${ name }_${ p.name }_unq_idx ON ${ plv8.quote_ident(name) } (${ p.name })`, 3);
			}
		});
		if( klass.indexes ){
			for(let k in klass.indexes){
				let idx = klass.indexes[k];
				let using = idx.using ? `USING ${ idx.using }` : '';
				alter(`CREATE ${ idx.unique ? 'UNIQUE' : '' } INDEX ${ name }_${ k }_idx ON ${ plv8.quote_ident(name) } ${ using } ( ${ idx.on.map(c => plv8.quote_ident(c) ).join(',') } )`, 3);
			}
		}
		let kp = klass.prototype;
		if( kp.beforeChange ){
			addListener('before', 'insert update', klass, kp.beforeChange);
		}
		if( kp.afterChange ){
			addListener('after','insert update', klass, kp.afterChange);
		}
		if( kp.beforeUpdate ){
			addListener('before','update', klass, kp.beforeUpdate);
		}
		if( kp.afterUpdate ){
			addListener('after','update', klass, kp.afterUpdate);
		}
		if( kp.beforeInsert ){
			addListener('before','insert', klass, kp.beforeInsert);
		}
		if( kp.afterInsert ){
			addListener('after','insert', klass, kp.afterInsert);
		}
		if( kp.beforeDelete ){
			addListener('before','delete', klass, kp.beforeDelete);
		}
		if( kp.afterDelete ){
			addListener('after','delete', klass, kp.afterDelete);
		}
	}

	function userValue(v, args) {
		if(v.type == 'ident'){
			return `$prev.${v.value}`;
		}
		if(v.type == 'placeholder'){
			if( v.value > args.length ){
				throw new QueryError({message: `missing argument for placeholder $${v.value}`});
			}
			console.info('$'+v.value, args[v.value-1], args);
			return plv8.quote_literal(args[v.value - 1]);
		}
		return plv8.quote_literal(v.value);
	}

	// variable interpolation
	// converts ["sql with $1 $2", arg1, arg2] => "sql with x y"
	function interpolate(a, cxtReverse){
		return a[0].replace(/\$(\d+)/g, function(match, ns){
			let n = parseInt(ns,10);
			if( n <= 0 ){
				throw new UserError(`invalid placeholder name: \$${ns}`);
			}
			if( a.length-1 < n ){
				throw new UserError(`no variable for placeholder: \$${ns}`);
			}
			if( typeof a[n] == 'undefined' ){
				throw new UserError(`placeholder variable ${n} is undefined`);
			}
			else if( typeof a[n] == 'boolean' ){
				return a[n].toString();
			}
			else if( typeof a[n] == 'number' ){
				return a[n].toString();
			}
			else if( a[n] instanceof Date ){
				return plv8.quote_literal(a[n]) + '::timestamptz';
			}
			else if( a[n] && cxtReverse[a[n]] ){
				return a[n];
			}
			return plv8.quote_literal(a[n]);
		});
	}

	// returns an SQL select with (single row single column)
	function sqlForProperty(klass, session, ast, vars=[], i=0){
		// fetch requested property
		let property = klass.props[ast.name];
		// if no property or query then assume it must be a simple
		// field included via sql 
		if( !property || !property.query ){
			if( ast.args.length > 0 ){
				err(`property type does not accept arguments`);
			}
			if( ast.filters.length > 0 ){
				err(`property type does not accept filters`);
			}
			return `$prev.${ast.name}`;
		}
		if( !property.type ){
			throw new QueryError({
				message: `property has no type defined`,
				property: ast.name,
				kind: klass.name,
			});
		}
		// shorthand for errors
		let err = function(msg){
			throw new QueryError({
				message: msg,
				property: property.name,
				type: property.type,
				kind: klass.name,
			});
		};
		// build context that can be used to reference parent table
		let cxtReverse = {};
		let cxt = Object.keys(klass.props).reduce(function(o, k){
			o[k] = '$prev.'+k;
			cxtReverse[o[k]] = true;
			return o;
		},{});
		cxt.session = session;
		// fetch the sql query
		let sql = property.query.apply(cxt, ast.args);
		// check if no query returned
		if( !sql ){
			err(`${klass.name}.${property.name} did not return a valid sql query`);
		}
		// check if we got an object back
		let shadow;
		if( !Array.isArray(sql) && typeof sql == 'object' ){
			let args = sql.args || [];
			// merge any withs from these types of queries into the main CTE to allow shadowing
			if( sql.with ){
				if( Array.isArray(sql.with) ){
					sql.with = sql.with.join(',');
				}
				if( typeof sql.with != 'string' ){
					err(`${klass.name}.${property.name} did not return a valid sql query object: expected 'with' to be a string`);
				}
				shadow = interpolate([sql.with].concat(args), cxtReverse);
			}
			// ensure query part is correct
			if( !sql.query || typeof sql.query != 'string' ){
				err(`${klass.name}.${property.name} did not return a valid sql query object: expected 'query' to be a string`);
			}
			// convert to array format
			sql = [sql.query].concat(sql.args);
		}
		// interpolate any variables into the sql to convert to string
		if( Array.isArray(sql) ){
			sql = interpolate(sql, cxtReverse);
		}
		if( typeof sql != 'string' ){
			err(`${klass.name}.${property.name} did not return a valid sql query: expected sql to be a string got: ${typeof sql}`);
		}
		let withs = [sql];
		let isArray = property.type == 'array';
		let format = isArray ? ARRAY_OF_SIMPLE : SIMPLE;
		// select the properties for subquery
		let targetKlass = schema[property.of || property.type];
		if( targetKlass ){
			let _ast = ast;
			let plucked = false;
			let halt = false;
			let originalProps = ast.props;
			let where = '';
			let order = '';
			// normalize ast filters
			// eg..
			//     my_property.pluck(x).pluck(y)
			// becomes...
			//     my_property.pluck(x.pluck(y))
			ast.filters = ast.filters.filter(function(f){
				if( halt ){
					err(`cannot use filter ${f.name} here`);
				}
				switch(f.name){
					case 'pluck':
						// push any chained plucks into the next property's filters
						if( plucked ){
							_ast.filters.push(f);
							plucked = f;
							return false;
						}
						if( _ast.props.length > 0 ){
							console.warn(`ignoring redundent ${property.name} selections ${_ast.props.map(p => p.name).join(',')} due to ${f.name}`);
						}
						_ast.props = [f.prop];
						_ast = f.prop;
						plucked = f;
						return true;
					case 'count':
						if( plucked ){
							console.warn(`ignoring redundent pluck filter on ${property.name} due to count`);
						}
						if( _ast.props.length > 0 ){
							console.warn(`ignoring redundent ${property.name} selections ${_ast.props.map(p => p.name).join(',')} due to ${f.name}`);
						}
						ast.props = [{
							name: 'id',
							args: [],
							props: [],
							filters: [],
						}];
						halt = true;
						return true;
					case 'where':
						if( where ){
							where += ' AND ';
						}
						let a = userValue(f.a, vars);
						let b = userValue(f.b, vars);
						where += `where ${a} ${f.op} ${b}`;
						return false;
					case 'sortBy':
						if( order ){
							err('multiple sort operations used');
						}
						order = `order by $prev.${f.ident} ${f.dir}`;
						return false;
					case 'sort':
						if( order ){
							err('multiple sort operations used');
						}
						let by = plucked ? 'plucked' : '$prev';
						order = `order by ${by} ${f.dir}`;
						return false;
					default:
						return true;
				}
			});
			// normalize property selection for plucks
			// eg..
			//     my_property.pluck(x).pluck(y){id}
			// becomes...
			//     my_property.pluck(x).pluck(y{id})
			if( plucked && originalProps.length > 0 ){
				plucked.prop.props = originalProps;
			}
			// If no properties explictly chosen assume ALL
			if( ast.props.length === 0 ){
				err(`expected at least one property selection`);
				ast.props = Object.keys(targetKlass).reduce(function(props, k){
					let p = targetKlass[k];
					if( !p.type ){
						return props;
					}
					props.push({
						name: k,
						args: [],
						props: [],
						filters: []
					});
					return props;
				},[]);
			}
			withs.unshift(`${sqlForClass(targetKlass, session, ast, vars, i+1)} from $prev ${where} ${order}`);
			format = isArray ? ARRAY_OF_OBJECTS : OBJECT;
		}
		// Add filter queries
		ast.filters.forEach(function(f){
			switch(f.name){
				case 'first':
					withs.unshift(`select * from $prev limit 1`);
					format = format == ARRAY_OF_OBJECTS ? OBJECT : SIMPLE;
					break;
				case 'pluck':
					format = ARRAY_OF_SIMPLE;
					break;
				case 'count':
					withs.unshift(`select count(*) from $prev`);
					format = SIMPLE;
					break;
				case 'take':
					withs.unshift(`select * from $prev limit ${f.n}`);
					break;
				case 'slice':
					withs.unshift(`select * from $prev offset ${f.start} limit ${f.end}`);
					break;
				case 'sort':
					withs.unshift(`select * from $prev order by $prev ${f.dir}`);
					break;
				case 'sortBy':
					withs.unshift(`select * from $prev order by $prev.${f.ident} ${f.dir}`);
					break;
				default:
					err(`unknown filter or cannot use filter here: '${f.name}'`);
			}
		});
		// convert sql to always return a single row with a single json column
		switch(format){
			case ARRAY_OF_SIMPLE: // multi row single col
				withs.unshift(`select coalesce(json_agg(to_json(row($prev.*))->'f1'),'[]'::json) from $prev`);
				break;
			case ARRAY_OF_OBJECTS: // multi row multi col
				withs.unshift(`select coalesce(json_agg($prev.*),'[]'::json) from $prev`);
				break;
			case OBJECT: // single row multi col
				withs.unshift(`select row_to_json($prev.*) from $prev`);
				break;
			case SIMPLE: // single row single col
				withs.unshift(`select to_json(row($prev.*))->'f1' from $prev`);
				break;
			default:
				err(`fatal: unexpected to-json format`);
		}
		let out = withs.reduce(function(w, sql, j){
			let curr = `q_${i}_${j}`;
			let prev = `q_${i}_${j+1}`;
			if( j < withs.length-1 ){ // ignore last (user sql)
				sql = sql.split('$prev').join(prev);
			}
			let ws = '\n' + Array(4*i).join(' ');
			if( j > 0 ){
				let comma = j > 1 ? ',' : '';
				return `${ws}${curr} as (${sql}) ${comma}` + w;
			}
			return `${ws}${sql}` + w;
		},'');
		if( shadow ){
			out = shadow + (withs.length>1 ? ',' : ' ') + out;
		}
		if( withs.length > 1 || shadow ){
			return `(with ${out})`;
		}else{
			return `(${out})`;
		}
	}

	// returns an SQL select with (multi row multi column)
	function sqlForClass(klass, session, ast, vars=[], i=0){
		let sql = 'select ' + ast.props.map(function(p){
			return sqlForProperty(klass, session, p, vars, i) + ` as ${p.alias}`;
		}).join(',');
		// if the class specifies a 'with' function then the sql will be
		// prepended with the 'with' statement
		if( klass.with ){
			if( typeof klass.with != 'function' ){
				throw new UserError(`'with' must be a function`);
			}
			let a = klass.with.apply(session, []);
			if( a ){
				if( typeof a == 'string' ){
					a = [a];
				}
				if( !Array.isArray(a) ){
					throw new UserError(`invalid return from 'with' function for ${klass.name}`);
				}
				let withSql = interpolate(a, {});
				sql = `with ${withSql} (${sql})`;
			}
		}
		return sql;
	}

	// hash of property signature
	// XXX: naive
	function sig(p){
		return JSON.stringify({
			name: p.name,
			filters: p.filters,
			args: p.args
		});
	}

	// deduplicates property selections
	function normalizeProps(props) {
		let seen = {};
		return props.filter(function(p){
			if( p.props.length > 0 ){
				p.props = normalizeProps(p.props);
			}
			let curr = {p:p, sig:sig(p)};
			let prev = seen[p.alias];
			if( prev ){
				// mismatched signatures
				if( prev.sig != curr.sig ){
					throw new QueryError({
						message:`conflicting properties named '${p.alias}'`,
					});
				}
				// merge
				mergeProps(prev.p.props, curr.p.props);
				// drop
				return false;
			}
			seen[p.alias] = curr;
			return true;
		});
	}

	function mergeProps(dest, src){
		src.forEach(function(sp){
			let found = false;
			dest.forEach(function(dp){
				if( sp.alias == dp.alias ){
					found = true;
					// check for conflicts
					if( sig(sp) != sig(dp) ){
						throw new QueryError({
							message:`conflicting properties for '${sp.alias}' cannot be merged`
						});
					}
					mergeProps(dp.props, sp.props);
				}
			});
			// if not found - add it
			if( !found ){
				dest.push(sp);
			}
		});
	}

	// Entity is the class to extend to define schema entities/tables.
	class Entity {
		static toString(){
			return this.name;
		}
		query(sql, ...args){
			return db.query(sql, ...args);
		}
		transaction(fn){
			return db.transaction(fn);
		}
	}
	arla.Entity = Entity;

	// trigger executes any before/after triggers defined on the entity
	arla.trigger = function(e){
		let op = e.opKind + '-' + e.op;
		['*', e.table].forEach(function(table){
			let ops = listeners[table];
			if( !ops || ops.length == 0){
				return;
			}
			let triggers = ops[op];
			if( !triggers ){
				return;
			}
			triggers.forEach(function(fn){
				let r = new fn.klass();
				for(let k in e.record){
					r[k] = e.record[k];
				}
				try{
					fn.apply(r,[e]);
				}catch(e){
					if(e.stack){
						console.debug(e.stack);
					}
					throw e;
				}
				for(let k in e.record){
					e.record[k] = r[k];
				}
			});
		});
		return e.record;
	};

	arla.replay = function(m){
		arla.exec(m, true);
		return true;
	};

	arla.exec = function(m, replay){
		if( !m.name ){
			throw new UserError('invalid action name');
		}
		if( !m.args ){
			m.args = [];
		}
		if( !m.version ){
			throw new MutationError({
				message: 'cannot exec mutation without version',
				mutation: m
			});
		}
		// if mutation is for an older version
		// ask the transform function to update it
		let iter = 0;
		while( m.version < arla.cfg.version ){
			// catch infinite recursion (ok 1000 isn't really infinite but if you
			// have 1000 versions you have bigger problems.)
			if( iter > 1000 ){
				throw new UserError('transform function appears to be causing infitie recursion');
			}
			if( !arla.cfg.transform ){
				throw new MutationError({
					message: `mutation requires transforming from version ${m.version} to ${arla.cfg.version} but no transform function was defined`,
					mutation: m
				});
			}
			console.debug(`transforming ${m.name} from ${m.version} to ${arla.cfg.version}...`);
			m = arla.cfg.transform(m, arla.cfg.version);
			iter++;
		}
		var fn = actions[m.name];
		if( !fn ){
			if( /^[a-zA-Z0-9_]+$/.test(m.name) ){
				throw new UserError(`no such action ${m.name}`);
			} else {
				throw new UserError('invalid action name');
			}
		}
		// exec the mutation func
		console.debug(`action ${m.name} args:`, m.args, 'session:', m.token);
		let cxt = {
			session:m.token,
			replay:replay,
			query: function(sql, ...args){
				return db.query(sql, ...args);
			},
			transaction: function(fn){
				return db.transaction(fn);
			},
		};
		var queryArgs;
		db.transaction(function(){
			queryArgs = fn.bind(cxt)(...m.args);
		});
		console.debug(`action ${m.name} returned`, queryArgs);
		if( !queryArgs ){
			return;
		}
		if( !Array.isArray(queryArgs) ){
			queryArgs = [queryArgs];
		}
		// ensure first arg is valid
		if( typeof queryArgs[0] != 'string' ){
			throw new UserError('invalid response from action. should be: [sqlstring, ...args]');
		}
		// run the query returned from the mutation func
		try{
			return db.query(...queryArgs);
		}catch(e){
			if(e.stack){
				console.debug(e.stack);
			}
			if( e.message ){
				if( (/violates unique constraint/i).test(e.message) ){
					e.message = 'violates unique constraint';
				}
			}
			throw new MutationError({
				message: e.message.replace('UserError: ', ''),
				mutation: m
			});
		}
	};

	arla.query = function({query, args, token}){
		if( !query ){
			throw new QueryError({error:'arla_query: query text cannot be null'});
		}
		query = `root(){ ${query} }`;
		console.debug('AQL:', query, args);
		let ast;
		try{
			ast = gql.parse(query);
		}catch(err){
			let e = err;
			if(e.stack){
				console.debug(e.stack);
			}
			if( e.line && e.offset ){
				console.warn( query.split(/\n/)[e.line-1] );
				console.warn( `${ Array(err.column).join('-') }^` );
				e = new QueryError({
					line: e.line,
					column: e.column,
					offset: e.offset,
					message: e.message,
					context: query
				});
			}else{
				e = new QueryError({
					message: e.message,
					context: query
				});
			}
			throw e;
		}
		// console.debug('AST (raw):', ast);
		ast.props = normalizeProps(ast.props);
		// console.debug('AST (normalized):', ast);
		if( ast.name != 'root' ){
			throw new QueryError({message:`expected root() property got ${ast.name}`});
		}
		let sql = sqlForClass(schema.root, token, ast, args);
		let res = db.query(sql)[0];
		// console.debug('RESULT', res);
		return res;
	};

	arla.authenticate = function(values){
		var res = db.query.apply(db, arla.cfg.authenticate(values));
		if( res.length < 1 ){
			throw new UserError('invalid credentials');
		}
		return res[0];
	};

	arla.register = function(values){
		return arla.cfg.register(values);
	};

	// init calls boostrap during app startup
	arla.bootstrap = function(stmts){
		if( !stmts ){
			return;
		}
		if( typeof stmts == 'string' ){
			stmts = [stmts];
		}
		if( !Array.isArray(stmts) ){
			throw new UserError(`bootstrap should be an array of SQL statements`);
		}
		stmts.forEach(function(stmt){
			let args = stmt;
			if( !Array.isArray(args) ){
				args = [args];
			}
			db.query.apply(db, args);
		});
	};

	// init will only ever run once on app startup
	arla.init = function(){
		ddl = ddl.sort(function(a,b){
			return a.priority - b.priority;
		});
		ddl.forEach(function(ddl){
			db.query(ddl.stmt);
		});
		arla.bootstrap(arla.cfg.bootstrap);
	};

	// configure will be run everytime a js context is started
	arla.configure = function(cfg){
		if( arla.cfg ){
			throw new Error('configure should only be called ONCE!');
		}
		arla.cfg = cfg;
		// setup user schema
		Object.keys(cfg.schema || {}).forEach(function(name){
			define(name, cfg.schema[name]);
		});
		// setup user actions
		Object.keys(cfg.actions || {}).forEach(function(name){
			action(name, cfg.actions[name]);
		});
		// validate some cfg options
		if( !cfg.authenticate ){
			throw new Error('missing required "authenticate" function');
		}
		if( !cfg.register ){
			throw new Error('missing required "register" function');
		}
		if( !cfg.version ){
			cfg.version = 1;
		}
	};

})();

// Execute the user's code
//CONFIG//
