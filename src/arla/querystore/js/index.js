import gql from "./graphql"

// UserError should be used when you want an error message to
// be shown to an end user.
class UserError extends Error {
  constructor(m) {
    var err = super(m);
    Object.assign(this, {
      name: "UserError",
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
    Object.assign(this, {
      name: "QueryError",
      message: JSON.stringify(o),
      stack: err.stack
    });
  }
}

(function(){

	var listeners = {};
	var tests = [];
	var ddl = [];
	var schema = {};
	var actions = {};

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

	function col({type = 'text', nullable = false, def = undefined, pk = false, onDelete = 'CASCADE', onUpdate = 'RESTRICT', ref} = {}) {
		if( type == 'timestamp' ){
			console.warn('there are issues with the timestamp type it is recordmend you use timestamptz');
		}
		if( !type && ref ){
			type = 'uuid';
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
		if( pk ){
			x.push('PRIMARY KEY')
		}
		if( def !== null ){
			x.push(`DEFAULT ${ def }`);
		}
		return x.join(' ');
	}

	function define(name, klass){
		klass.name = name;
		if( !klass.id ){
			klass.id = {type:'uuid', pk:true, def:'uuid_generate_v4()'}
		}
		let columns = Object.keys(klass).reduce(function(props, k){
			if(!klass[k].type){
				return props;
			}
			klass[k].name = k;
			klass[k].klass = klass;
			if(klass[k].query){
        // add the magic _count property
        if( klass[k].type == 'array' && !klass[k+'_count']){
          klass[k+'_count'] = {
            name: k+'_count',
            klass: klass,
            type: 'counter',
            queryName: k
          }
        }
				return props;
			}
			props.push(klass[k]);
			return props;
		},[]);

		let alter = function(stmt){
			if(name != 'root'){
				ddl.push(stmt)
			}
		}
		alter(`CREATE TABLE ${ plv8.quote_ident(name) } ()`);
		alter(`CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('before')`);
		alter(`CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON ${ plv8.quote_ident(name) } DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('after')`);
		columns.forEach(function(p){
			alter(`ALTER TABLE ${ plv8.quote_ident(name) } ADD COLUMN ${ plv8.quote_ident(p.name) } ${ col(p) }`);
			if(p.unique){
				alter(`CREATE UNIQUE INDEX ${ name }_${ p.name }_unq_idx ON ${ plv8.quote_ident(name) } (${ p.name })`)
			}
		})
		if( klass.indexes ){
			for(let k in klass.indexes){
				let idx = klass.indexes[k];
				let using = idx.using ? `USING ${ idx.using }` : '';
				alter(`CREATE ${ idx.unique ? 'UNIQUE' : '' } INDEX ${ name }_${ k }_idx ON ${ plv8.quote_ident(name) } ${ using } ( ${ idx.on.map(c => plv8.quote_ident(c) ).join(',') } )` );
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
		schema[name] = klass;
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

	function sqlForProperty(klass, session, ast, i=0){
    let alias = ast.alias || ast.name;
		let property = klass[ast.name];
		if( !property ){
			throw new UserError(`${klass.name} has no property ${ast.name}`)
		}
		if( !property.type ){
			throw new UserError(`${klass.name} does not have a valid property ${ast.name}`)
		}
    // if this is a counter then get the _real_ property
    let counter = null;
    if( property.type == 'counter' ){
      counter = property;
      property = klass[property.queryName];
    }
		// simple property fetch
		if( !property.query ){
			return `$prev.${property.name}`;
		}
		// build context that can be used to reference parent table
		let cxtReverse = {};
		let cxt = Object.keys(klass).reduce(function(o, k){
			o[k] = '$prev.'+k;
			cxtReverse[o[k]] = true;
			return o;
		},{});
		cxt.session = session;
		// fetch the sql query
		let sql = property.query.apply(cxt, ast.args);
		// interpolate any variables into the sql
		if( Array.isArray(sql) ){
			sql = sql[0].replace(/\$(\d+)/g, function(match, ns){
				let n = parseInt(ns,10);
				if( n <= 0 ){
					throw new UserError(`invalid placeholder name: \$${ns}`);
				}
				if( sql.length-1 < n ){
					throw new UserError(`no variable for placeholder: \$${ns}`);
				}
				if( typeof sql[n] == 'undefined' ){
					console.warn(`placeholder variable ${n} is undefined in query for ${klass.name} ${property.name}`);
				}
				if( sql[n] && cxtReverse[sql[n]] ){
					return sql[n];
				}
				return plv8.quote_literal(sql[n]);
			});
		}
		// no query returned
		if( !sql ){
			throw new UserError(`no query sql returned from edge call: ${property.name} on ${klass.name}`);
		}
		let jsonfn = 'row_to_json';
		let jsondef = '{}';
		let type = property.type;
    if( ast.filters.first ){
      if( type != 'array' ){
        throw QueryError({
          message: `cannot use filter 'first' on non-array property`,
          property: property.name,
          type: property.type,
          kind: klass.name,
        })
      }
      ast.filters.limit = 1;
      type = property.of;
    }
		if( type == 'array' ){
			jsonfn = 'json_agg';
			jsondef = '[]';
			type = property.of;
			if( !type ){
				throw new UserError(`${klass.name} ${property.name} declares an array type but without an 'of' type set`);
			}
		} else if( ['int', 'integer', 'text', 'uuid'].indexOf(type) >= 0 ){
      return `(${sql}) as ${alias}`;
    }
    let withs = [sql];
    if( ast.filters.limit ){
      withs.unshift(`select * from $prev limit ${plv8.quote_literal(ast.filters.limit)}`)
    }
    if( counter ){
      withs.unshift(`select count(*) from $prev`);
    }else{
      if( schema[type] ){
        withs.unshift(`${sqlForClass(schema[type], session, ast, i+1)} from $prev`);
      }
      withs.unshift(`select coalesce(${jsonfn}($prev.*),'${jsondef}'::json) from $prev`)
    }
    return '(with' + withs.reduce(function(w, sql, j){
      let curr = `q_${i}_${j}`;
      let prev = `q_${i}_${j+1}`;
      if( j < withs.length-1 ){ // ignore last (user sql)
        sql = sql.split('$prev').join(prev)
      }
      if( j > 0 ){
        let comma = j > 1 ? ',' : '';
        return `\n${curr} as (${sql}) ${comma}` + w;
      }
      return `\n${sql}` + w;
    },'') + `\n) as ${alias}`;
	}

	function sqlForClass(klass, session, ast, i = 0){
		if(ast.kind == 'property'){
			throw new UserError('invalid query: expected property definitions');
		}
		return "select " + Object.keys(ast.props).map(function(k){
			return sqlForProperty(klass, session, ast.props[k], i)
		}).join(',');
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
				let r = new fn.klass();
				for(var k in e.record){
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
				for(var k in e.record){
					e.record[k] = r[k];
				}
			});
		});
		return e.record;
	};

	arla.replay = function(m){
		arla.exec(m.Name, m.Token, m.Args);
		return true;
	};

	arla.exec = function(name, session, args){
		var fn = actions[name];
		if( !fn ){
			if( /^[a-zA-Z0-9_]+$/.test(name) ){
				throw new Error(`no such action ${name}`);
			} else {
				throw new Error('invalid action');
			}
		}
		// exec the mutation func
		console.debug(`action ${name} args:`, args, "session:", session);
		var queryArgs = fn.apply({session:session}, args);
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
			throw new Error('invalid response from action. should be: [sqlstring, ...args]');
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
					throw new UserError("violates unique constraint");
				}
			}
			throw e;
		}
	};

	arla.query = function(session, query){
		if( !query ){
			throw new QueryError({error:'arla_query: query text cannot be null'});
		}
		query = `root(){ ${query} }`;
		console.debug("QUERY:", query);
		let ast;
		try{
			ast = gql.parse(query);
		}catch(err){
			if( err.line && err.offset ){
				console.warn( query.split(/\n/)[err.line-1] );
				console.warn( `${ Array(err.column).join('-') }^` );
				err = new QueryError({
          line: err.line,
          column: err.column,
          offset: err.offset,
          error: err.message,
          context: query
        });
			}
			throw err;
		}
		console.debug("AST:", ast);
		let sql = sqlForClass(schema.root, session, ast[0]);
		let res = db.query(sql)[0];
		console.debug("RESULT", res);
		return res;
	};

	arla.authenticate = function(values){
		var res = db.query.apply(db, arla.cfg.authenticate(values));
		if( res.length < 1 ){
			throw new Error('invalid credentials');
		}
		return res[0];
	}

	arla.register = function(values){
		return arla.cfg.register(values);
	}

	// init will only ever run once
	arla.init = function(){
		ddl.forEach(function(stmt){
			db.query(stmt);
		});
	}

	// configure will be run everytime a js context is started
	arla.configure = function(cfg){
		if( arla.cfg ){
			throw new Error('configure should only be called ONCE!');
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
		// store cfg for later
		arla.cfg = cfg;
		// validate some cfg options
		if( !arla.cfg.authenticate ){
			throw new Error('missing required "authenticate" function');
		}
		if( !arla.cfg.register ){
			throw new Error('missing required "register" function');
		}
	}

})();

// Execute the user's code
//CONFIG//
