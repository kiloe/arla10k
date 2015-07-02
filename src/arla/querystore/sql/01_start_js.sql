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
	plv8.arla = arla;

	// add console logging
	var console = (function(console){

		console.ALL = 0;
		console.DEBUG = 1;
		console.INFO = 2;
		console.LOG = 3;
		console.WARN = 4;
		console.ERROR = 5;
		console.logLevel = console.INFO;
		function logger(level, pglevel, tag) {
			var args = [];
			for (var i = 2; i < arguments.length; i++) {
				args.push(arguments[i]);
			}
			var msg = args.map(function(msg){
				if( typeof msg == 'object' ){
					msg = JSON.stringify(msg, null, 4);
				}
				return msg;
			}).join(' ');
			(msg || '').split(/\n/g).forEach(function(line){
				if( console.logLevel <= level ){
					plv8.elog(pglevel, line);
				}
			})
		}
		console.debug = logger.bind(console, console.DEBUG, NOTICE, "DEBUG:");
		console.info  = logger.bind(console, console.INFO, NOTICE, "INFO:");
		console.log   = logger.bind(console, console.LOG, NOTICE, "LOG:");
		console.warn  = logger.bind(console, console.WARN, NOTICE, "WARN:");
		console.error = logger.bind(console, console.ERROR, NOTICE, "ERROR:");
		return console;
	})({});

	// expose database
	var db = (function(db){
		db.query = function(sql){
			var args = [];
			for (var i = 1; i < arguments.length; i++) {
				args.push(arguments[i]);
			}
			console.debug(sql, args);
			return plv8.execute(sql, args);
		}
		db.transaction = plv8.subtransaction;
		return db;
	})({});
