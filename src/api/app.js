var http = require('http');
var pg = require('pg');
var jwt = require('jwt-simple');
var store = require('./store');
var ident = require('./ident');
var express = require('express');
var path = require('path');
var multer = require('multer');
var bodyParser = require('body-parser');
var errors = require('./errors');
var moment = require('moment');
var Promise = require('es6-promise').Promise;
var uuid = require('node-uuid');

module.exports = function(opts){

	// Options
	opts = opts || {};

	// Default datadir
	if( !opts.dataDir ){
		opts.dataDir = '/var/lib/arla/data';
	}

	// Set the secret that will be used for auth tokens
	var secret = opts.secret || process.env.AUTH_SECRET;
	if( !secret ){
		console.error('required environment variable AUTH_SECRET was not set!');
		console.error('without a secret the server cannot read auth tokens');
		process.exit(1);
	}

	// Create the main application
	var app =  express();

	// port to listen on
	app.port = opts.port || 3000;

	// static assets path
	app.assetPath = opts.assetPath || process.env.APP_PATH || "/var/lib/arla/app/public"
	app.use(express.static(app.assetPath));

	// Postgres connection
	app.conString = opts.conString || process.env.QUERY_DATABASE || "socket:/var/run/postgresql/?encoding=utf8";

	// Enable debug
	app.debug = process.env.DEBUG=="true";

	// Set email address of superuser
	app.su = opts.su || process.env.AUTH_SU;

	// The passwd file
	app.passwd = ident.open(path.join(opts.dataDir, 'passwd.db'));

	// The write ahead log of data mutations
	app.wal = store.open(path.join(opts.dataDir, 'datastore.db'));

	// Enable parsing plain text
	app.use('/query', bodyParser.text());

	// Enable parsing application/json requests
	app.use(bodyParser.json()); // for parsing application/json

	// Enable parsing application/x-www-form-urlencoded requests
	app.use(bodyParser.urlencoded({ extended: true }));

	// enable multipart/form-data (uploads) requests
	app.use(require('multer')({
		dest: '/tmp',
		limits: {
			fileSize: 1024*1024
		}
	}));

	// All responses are either SUCCESS (200), FAIL (400), UNAUTH (403) or FATAL (500)
	app.use(function(req, res, next){
		res.ok = function(o){
			res.json(o || {});
			if( app.debug ){
				console.log('response ok:', JSON.stringify(o,null,4));
			}
		}
		res.fail = function(err){
			var errs = err;
			if( !Array.isArray(errs) ){
				errs = [errs];
			}
			if( app.debug ){
				errs.forEach(function(err){
					if( err.stack ){
						console.error(err.stack);
						if( err.detail ){
							console.error(err.detail);
						}
						if( err.hint ){
							console.error(err.hint);
						}
					} else {
						console.error(err.toString());
					}
				});
			}
			var st;
			switch(err){
				case errors.InvalidUserId = 'invalid user id':
				case errors.InvalidPassword = 'invalid password':
				case errors.InvalidToken = 'invalid token':
				case errors.TokenExpired = 'token expired':
					st = 403;
					break;
				default:
					st = 400;
			}
			res.status(st).json({
				errors: errs.map(function(e){
					return e.toString()
				})
			});
		}
		next();
	})

	app.query = function(sql, args){
		return new Promise(function(resolve, reject){
			pg.connect(app.conString, function(err, db, done){
				if( err ){
					done(db);
					return reject(err);
				}
				db.query(sql, args, function(err, result){
					done(db);
					if( err ){
						return reject(err);
					}
					resolve(result.rows);
				})
			})
		})
	}

	// Grab the user id from jwt token
	app.use(['/exec', '/query'], function(req, res, next){
		var token = (req.headers.authorization || '').trim().replace(/^Bearer\s+/i,'');
		if( !token ){
			return res.fail('missing required Authorization header');
		}
		try{
			req.uid = jwt.decode(token, secret).id;
		}catch(err){
			return res.fail('invalid token');
		}
		if( !req.uid ){
			return res.fail('invalid user id');
		}
		next();
	});

	// Unhandled exception handler
	app.use(function(err, req, res, next){
		console.error('Unhandled exception:', err.stack);
		res.status(500).json({errors: ['there was a problem processing the request']});
		next(err);
	})

	// Function to run on startup that syncs data with query-engine
	app.sync = function(){
		return app.query('select * from meta').then(function(rows){
			var qinfo = rows.reduce(function(o, row){
				o[ row.key ] = row.value;
				return o;
			}, {});
			return app.wal.info().then(function(sinfo){
				var stream = null;
				var count = 0; // No. of ops stremed
				if( !sinfo.store_id ){
					return Promise.reject(Error('datastore did not have a valid store_id'));
				}
				if( !qinfo.store_id ){
					// query engine is empty... set store_id and stream from start
					stream = app.query("insert into meta (key,value) values ('store_id', $1)", [sinfo.store_id]).then(function(){
						return app.wal.stream();
					})
				} else if( qinfo.store_id != sinfo.store_id ){
					// query engine was previously populated from another data source
					return Promise.reject(Error('cannot sync data source with query engine: store_id mismatch'));
				} else {
					// stream after last known point
					stream = app.wal.stream(qinfo.last_id)
				}
				// stream each value to query engine
				var p = Promise.resolve();
				return stream.then(function(stream){
					return new Promise(function(resolve, reject){
						stream.onValue(function(o){
							count++;
							p = p.then(function(){
								return exec(o.value[0], o.value[1], true)
							});
						}).onEnd(function(){
							resolve(p);
						}).onError(reject);
					})
				})
			})
		})
	}

	// Start the server
	app.start = function(){
		return app.sync().then(function(){
			return new Promise(function(resolve, reject){
				var server = app.listen(app.port, function(){
					resolve(app);
				})
			})
		})
	}

	// Normalize an email/nickname/id to a uuid
	function find(id){
		if( /^[0-9a-f]{22}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id) ){
			return Promise.resolve(id);
		} else {
			return exec('find_member', [id]).then(function(id){
				if( !id ){
					throw errors.InvalidUserId;
				}
				return id
			})
		}
		return Promise.reject(errors.InvalidUserId);
	}

	function authByToken(token){
		return new Promise(function(resolve, reject){
			var t;
			try {
				t = jwt.decode(token, secret);
			} catch(err) {
				return reject(errors.InvalidToken);
			}
			if( t.type != 'refresh' ){
				return reject(errors.InvalidToken);
			}
			if( moment(t.exp).isBefore( moment() ) ){
				return reject(errors.TokenExpired);
			}
			resolve(app.passwd.validateId(t.id));
		})
	}

	function authByUserPass(username, password){
		return find(username).then(function(id){
			return app.passwd.validatePassword(id, password);
		})
	}

	function auth(o){
		if( o.token ){
			return authByToken(o.token);
		}
		return authByUserPass(o.username, o.password);
	}

	function ghost(id, as){
		if( app.su == id && as ){
			return find(as)
		}
		return id;
	}

	function authenticationHandler(req, res){
		return auth(req.body).then(function(id){
			return ghost(id, req.body.as)
		}).then(function(id){
			res.ok({
				access_token: jwt.encode({
					type: 'access',
					su: app.su == id,
					id: id,
					exp: moment().add(2, 'hours')
				}, secret),
				refresh_token: jwt.encode({
					type: 'refresh',
					id: id,
					exp: moment().add(1, 'day')
				}, secret)
			});
		}).catch(function(err){
			res.fail(err);
		})
	}

	// Execute an action on the data
	function exec(name, args, doNotLog){
		return app.query('select arla_exec($1::text, $2::json) as v', [name, JSON.stringify(args)]).then(function(rows){
			var v = rows && rows.length > 0 ? rows[0].v : null;
			var o = [name, args]; // action name, args;
			return doNotLog ? v : app.wal.put(o).then(function(){
				return v;
			})
		});
	}

	// Handler for executing actions
	function execHandler(req, res){
		var o = req.body;
		return exec(o.name, o.args).then(function(){
			res.ok();
		}).catch(function(err){
			res.fail(err);
		})
	}

	// Handler for graphql-like queries
	function queryHandler(req, res){
		var q = 'member('+req.uid+'){' + req.body + '}';
		return app.query('select arla_query($1::text) as res', [q]).then(function(rows){
			if( rows.length == 0 ){
				return res.fail('no rows returned');
			}
			if( !rows[0].res ){
				return res.fail('unexpected response from query');
			}
			// if nothing at all in the response this likely means
			// that while we trust the token, it is no longer referring to
			// a member in the query store
			if( !rows[0].res.member ){
				return res.fail(errors.InvalidToken);
			}
			res.ok(rows[0].res.member);
		}).catch(function(err){
			res.fail(err)
		});
	}

	// User registration handler
	function registrationHander(req, res){
		var u = req.body;
		if( !u.id ){
			u.id = uuid.v4();
		}
		return app.passwd.set(u.id, u.password).then(function(){
			delete u.password;
			return exec('create_member', [u])
		}).then(function(){
			res.ok({
				access_token: jwt.encode({
					type: 'pending',
					id: u.id,
					exp: moment().add(3, 'days')
				}, secret)
			})
		}).catch(function(err){
			res.fail(err);
		})
	}

	app.post('/register', registrationHander);
	app.post('/auth', authenticationHandler);
	app.post('/exec', execHandler);
	app.post('/query', queryHandler);

	return app;

}
