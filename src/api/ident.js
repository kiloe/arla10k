
var sqlite3 = require('sqlite3');
var Promise = require('es6-promise').Promise;
var moment = require('moment');
var bcrypt = require('bcrypt');
var errors = require('./errors');

var Store = function(path, opts){
	this.path = path;
	if( !path ){
		throw Error("path augument is required");
	}
	this.opts = opts || {};
	this.failures = (function(){
		var data = {};
		var store = {
			expire: {minutes: 10}
		};
		
		store.collectGarbage = function(){
			var expiry = moment().subtract(store.expire);
			for(var k in data){
				data[k] = data[k].filter(function(time){
					return time.isAfter(expiry)
				});
				if( data[k].length === 0 ){
					delete data[k];
				}
			}
		}

		store.add = function(id){
			if( !data[id] ){
				data[id] = [];
			}
			data[id].push( moment() );
		}

		store.count = function(id){
			var fails = data[id];
			if( !fails ){
				return 0;
			}
			return fails.length;
		}

		setInterval(store.collectGarbage, 1000*60*5);

		return store;
	})();
	this.ready = this.open().catch(function(err){
		console.error(err);
		process.exit(1);
	});
};

Store.prototype.open = function(){
	var store = this;
	if( store.db ){
		return Promise.resolve().then(function(){
			return store;
		})
	}
	return new Promise(function(resolve, reject){
		store.db = new sqlite3.Database(store.path, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE, function(err){
			if( err ){
				reject(err);
				return;
			}
			store.db.serialize(function() {
				store.db.run("CREATE TABLE IF NOT EXISTS password (id TEXT UNIQUE, hash TEXT)", [], function(err){
					if( err ){
						reject(err);
						return;
					}
					resolve(store);
				});
			});
		});
	})
}

// Hash and store a password
Store.prototype.set = function(id, password){
	var store = this;
	return store.ready.then(function(){
		return store._set(id, password);
	})
}

Store.prototype._set = function(id, password){
	var store = this;
	return new Promise(function(resolve, reject){
		var issues = passwordWeeknesses(password);
		if( issues.length > 0 ){
			return reject(issues);
		}
		bcrypt.hash(password, 10, function(err, hash){
			if(err){
				return reject(err);
			}
			store.db.serialize(function(){
				store.db.run("INSERT INTO password (id, hash) VALUES (?, ?)", [id, hash], function(err){
					if( err ){
						reject(err);
						return;
					}
					resolve(id);
				});
			});
		});
	});
}

// Validate id
Store.prototype.validateId = function(id){
	var store = this;
	return store.ready.then(function(){
		return store._validateId(id);
	})
}

Store.prototype._validateId = function(id){
	var store = this;
	return new Promise(function(resolve, reject){
		store.db.serialize(function(){
			store.db.run("SELECT id FROM password WHERE id = ?", [id], function(err){
				if( err ){
					reject(err);
					return;
				}
				if( store.failures.count(id) > 10 ){
					reject(errors.LockedUserId);
					return;
				}
				resolve(id);
			});
		})
	})
}

// Validate password
Store.prototype.validatePassword = function(id, password){
	var store = this;
	return store.ready.then(function(){
		return store._validatePassword(id, password);
	})
}

Store.prototype._validatePassword = function(id, password){
	var store = this;
	return new Promise(function(resolve, reject){
		store.db.serialize(function(){
			store.db.get("SELECT hash FROM password WHERE id = ?", [id], function(err, row){
				if( err ){
					reject(err);
					return;
				}
				if( !row || !row.hash){
					reject(errors.InvalidUserId);
					return;
				}
				var hash = row.hash;
				bcrypt.compare(password, hash, function(err, ok){
					if( err ){
						reject(errors.InvalidPassword);
						return;
					}
					if( !ok ){
						store.failures.add(id);
						reject(errors.InvalidPassword);
						return;
					}
					if( store.failures.count(id) > 10 ){
						reject(errors.LockedUserId);
						return;
					}
					resolve(id);
				})
			});
		})
	})
}

function passwordWeeknesses(pw){
	if( !pw ){
		pw = '';
	}
	return [
		[errors.PasswordTooShort, pw.length < 9],
		[errors.PasswordTooNumeric, /^[0-9]+$/.test(pw) ],
		[errors.PasswordTooSimple, pw.length < 16 && /^([a-z]+|[A-Z]+)$/.test(pw)],
	].filter( function(x){
		return x[1];
	}).map( function(x){
		return x[0];
	});
}

exports.open = function(path, opts){
	return new Store(path, opts);
}
