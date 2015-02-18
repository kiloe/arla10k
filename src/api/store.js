var sqlite3 = require('sqlite3');
var Kefir = require('kefir');
var Promise = require('es6-promise').Promise;
var moment = require('moment');
var uuid = require('node-uuid');

var Store = function(path, opts){
	this.path = path;
	if( !path ){
		throw Error("path augument is required");
	}
	this.opts = opts || {};
	this.ready = this.open();
};

Store.prototype.open = function(){
	var store = this;
	if( store.opened ){
		return Promise.resolve(store);
	}
	store.opened = true;
	return new Promise(function(resolve, reject){
		store.db = new sqlite3.Database(store.path, sqlite3.OPEN_READWRITE|sqlite3.OPEN_CREATE, function(err){
			if( err ){
				reject(err);
				return;
			}
			store.db.serialize(function() {
				store.db.run("CREATE TABLE IF NOT EXISTS log (id INTEGER PRIMARY KEY AUTOINCREMENT, at INTEGER DEFAULT (datetime('now', 'localtime')), value TEXT)", [], function(err){
					if( err ){
						return reject(err);
					}
					store.db.run("CREATE TABLE IF NOT EXISTS meta (key TEXT NOT NULL UNIQUE, value TEXT NOT NULL)", [], function(err){
						if( err ){
							return reject(err);
						}
						store.db.all("SELECT value FROM  meta WHERE key = 'store_id'", [], function(err, rows){
							if( err ){
								return reject(err);
							}
							if( rows && rows.length > 0 ){
								return resolve(store);
							}
							store.db.run("INSERT INTO meta (key,value) VALUES ('store_id',?)", [uuid.v4()], function(err){
								if( err ){
									return reject(err);
								}
								resolve(store);
							})
						})
					})
				});
			});
		});
	})
}

// Commit the given object to the log
Store.prototype.put = function(o){
	var store = this;
	return store.ready.then(function(){
		return store._put(o);
	})
}

Store.prototype._put = function(o){
	var store = this;
	return new Promise(function(resolve, reject){
		store.db.serialize(function(){
			store.db.run("INSERT INTO log (value) VALUES (?)", [JSON.stringify(o)], function(err){
				if( err ){
					return reject(err);
				}
				resolve(this.lastID);
			});
		})
	});
}

// Fetch meta data
Store.prototype.info = function(){
	var store = this;
	return store.ready.then(function(){
		return store._info();
	})
}

Store.prototype._info = function(o){
	var store = this;
	return new Promise(function(resolve, reject){
		store.db.serialize(function(){
			store.db.all("SELECT * FROM meta", [], function(err, rows){
				if( err ){
					return reject(err);
				}
				resolve(rows.reduce(function(o, row){
					o[ row.key ] = row.value;
					return o;
				}, {}));
			});
		})
	});
}

// Return a stream of all objectes after the given id.
// If no id is given then returns a stream from the start.
Store.prototype.stream = function(afterId){
	var store = this;
	return store.ready.then(function(){
		return store._stream(afterId);
	})
}
Store.prototype._stream = function(afterId){
	var store = this;
	var emitter = Kefir.emitter();
	var sql = "SELECT id,at,value FROM log"
	var args = [];
	if( afterId ){
		sql += " WHERE id > ?";
		args.push(afterId);
	}
	store.db.each(sql + " ORDER BY id ASC", [], function(err, row){
		if( err ){
			emitter.error(err);
			return;
		}
		emitter.emit({
			id: row.id,
			at: moment(row.at),
			value: JSON.parse(row.value)
		});
	}, function(err){
		if( err ){
			emitter.error(err);
		}
		emitter.end();
	});
	return emitter;
}

exports.open = function(path, opts){
	return new Store(path, opts);
}
