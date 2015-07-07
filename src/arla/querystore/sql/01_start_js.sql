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

	// Production steps of ECMA-262, Edition 6, 22.1.2.1
	// Reference: https://people.mozilla.org/~jorendorff/es6-draft.html#sec-array.from
	if (!Array.from) {
	  Array.from = (function () {
	    var toStr = Object.prototype.toString;
	    var isCallable = function (fn) {
	      return typeof fn === 'function' || toStr.call(fn) === '[object Function]';
	    };
	    var toInteger = function (value) {
	      var number = Number(value);
	      if (isNaN(number)) { return 0; }
	      if (number === 0 || !isFinite(number)) { return number; }
	      return (number > 0 ? 1 : -1) * Math.floor(Math.abs(number));
	    };
	    var maxSafeInteger = Math.pow(2, 53) - 1;
	    var toLength = function (value) {
	      var len = toInteger(value);
	      return Math.min(Math.max(len, 0), maxSafeInteger);
	    };

	    // The length property of the from method is 1.
	    return function from(arrayLike/*, mapFn, thisArg */) {
	      // 1. Let C be the this value.
	      var C = this;

	      // 2. Let items be ToObject(arrayLike).
	      var items = Object(arrayLike);

	      // 3. ReturnIfAbrupt(items).
	      if (arrayLike == null) {
	        throw new TypeError("Array.from requires an array-like object - not null or undefined");
	      }

	      // 4. If mapfn is undefined, then let mapping be false.
	      var mapFn = arguments.length > 1 ? arguments[1] : void undefined;
	      var T;
	      if (typeof mapFn !== 'undefined') {
	        // 5. else
	        // 5. a If IsCallable(mapfn) is false, throw a TypeError exception.
	        if (!isCallable(mapFn)) {
	          throw new TypeError('Array.from: when provided, the second argument must be a function');
	        }

	        // 5. b. If thisArg was supplied, let T be thisArg; else let T be undefined.
	        if (arguments.length > 2) {
	          T = arguments[2];
	        }
	      }

	      // 10. Let lenValue be Get(items, "length").
	      // 11. Let len be ToLength(lenValue).
	      var len = toLength(items.length);

	      // 13. If IsConstructor(C) is true, then
	      // 13. a. Let A be the result of calling the [[Construct]] internal method of C with an argument list containing the single item len.
	      // 14. a. Else, Let A be ArrayCreate(len).
	      var A = isCallable(C) ? Object(new C(len)) : new Array(len);

	      // 16. Let k be 0.
	      var k = 0;
	      // 17. Repeat, while k < len… (also steps a - h)
	      var kValue;
	      while (k < len) {
	        kValue = items[k];
	        if (mapFn) {
	          A[k] = typeof T === 'undefined' ? mapFn(kValue, k) : mapFn.call(T, kValue, k);
	        } else {
	          A[k] = kValue;
	        }
	        k += 1;
	      }
	      // 18. Let putStatus be Put(A, "length", len, true).
	      A.length = len;
	      // 20. Return A.
	      return A;
	    };
	  }());
	}

	// arla global
	var arla = {};
	plv8.arla = arla;

	// add console logging
	var console = (function(console){

		console.UNKNOWN = 0;
		console.ALL = 1;
		console.DEBUG = 2;
		console.INFO = 3;
		console.LOG = 4;
		console.WARN = 5;
		console.ERROR = 6;
		console.logLevel = console.ALL;
		function logger(level, pglevel, tag) {
			if( level < console.logLevel ){
				return;
			}
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
				plv8.elog(pglevel, line);
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

	// expose a password hashing function
	var pgcrypto = (function(pgcrypto){
		pgcrypto.crypt = function(pw){
			if( !pw ){
				throw new Error('must supply a password to crypt');
			}
			var res = db.query("select crypt($1, gen_salt('bf')) as v", pw);
			if( res.length < 1 ){
				throw new Error('invalid response from crypt');
			}
			if( !res[0].v ){
				throw new Error('unexpected response from crypt');
			}
			return res[0].v;
		}
		return pgcrypto;
	})({});
