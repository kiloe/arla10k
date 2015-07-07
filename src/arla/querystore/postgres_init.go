package querystore
const postgresInitScript = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "plv8";
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
(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
"use strict";

module.exports = (function () {
  /*
   * Generated by PEG.js 0.8.0.
   *
   * http://pegjs.majda.cz/
   */

  function peg$subclass(child, parent) {
    function ctor() {
      this.constructor = child;
    }
    ctor.prototype = parent.prototype;
    child.prototype = new ctor();
  }

  function SyntaxError(message, expected, found, offset, line, column) {
    this.message = message;
    this.expected = expected;
    this.found = found;
    this.offset = offset;
    this.line = line;
    this.column = column;

    this.name = "SyntaxError";
  }

  peg$subclass(SyntaxError, Error);

  function parse(input) {
    var options = arguments.length > 1 ? arguments[1] : {},
        peg$FAILED = {},
        peg$startRuleFunctions = { start: peg$parsestart },
        peg$startRuleFunction = peg$parsestart,
        peg$c0 = peg$FAILED,
        peg$c1 = [],
        peg$c2 = function peg$c2(c) {
      return c;
    },
        peg$c3 = function peg$c3(expr, props) {
      return expression(expr, props);
    },
        peg$c4 = ".",
        peg$c5 = { type: "literal", value: ".", description: "\".\"" },
        peg$c6 = function peg$c6(p) {
      return p;
    },
        peg$c7 = /^[a-zA-Z0-9_]/,
        peg$c8 = { type: "class", value: "[a-zA-Z0-9_]", description: "[a-zA-Z0-9_]" },
        peg$c9 = null,
        peg$c10 = function peg$c10(name, args) {
      return [name, args];
    },
        peg$c11 = "(",
        peg$c12 = { type: "literal", value: "(", description: "\"(\"" },
        peg$c13 = ")",
        peg$c14 = { type: "literal", value: ")", description: "\")\"" },
        peg$c15 = function peg$c15(args) {
      return args;
    },
        peg$c16 = ",",
        peg$c17 = { type: "literal", value: ",", description: "\",\"" },
        peg$c18 = function peg$c18(arg) {
      return arg;
    },
        peg$c19 = "'",
        peg$c20 = { type: "literal", value: "'", description: "\"'\"" },
        peg$c21 = function peg$c21(a) {
      return a;
    },
        peg$c22 = /^[a-zA-Z0-9!=>\-< \t]/,
        peg$c23 = { type: "class", value: "[a-zA-Z0-9!=>\\-< \\t]", description: "[a-zA-Z0-9!=>\\-< \\t]" },
        peg$c24 = function peg$c24(a) {
      return a.trim();
    },
        peg$c25 = function peg$c25(p) {
      return p;
    },
        peg$c26 = /^[a-zA-Z_0-9]/,
        peg$c27 = { type: "class", value: "[a-zA-Z_0-9]", description: "[a-zA-Z_0-9]" },
        peg$c28 = function peg$c28(name) {
      return { kind: "property", name: name };
    },
        peg$c29 = "{",
        peg$c30 = { type: "literal", value: "{", description: "\"{\"" },
        peg$c31 = "}",
        peg$c32 = { type: "literal", value: "}", description: "\"}\"" },
        peg$c33 = /^[ \t\n]/,
        peg$c34 = { type: "class", value: "[ \\t\\n]", description: "[ \\t\\n]" },
        peg$c35 = { type: "other", description: "integer" },
        peg$c36 = /^[0-9]/,
        peg$c37 = { type: "class", value: "[0-9]", description: "[0-9]" },
        peg$c38 = function peg$c38(digits) {
      return parseInt(digits.join(""), 10);
    },
        peg$currPos = 0,
        peg$reportedPos = 0,
        peg$cachedPos = 0,
        peg$cachedPosDetails = { line: 1, column: 1, seenCR: false },
        peg$maxFailPos = 0,
        peg$maxFailExpected = [],
        peg$silentFails = 0,
        peg$result;

    if ("startRule" in options) {
      if (!(options.startRule in peg$startRuleFunctions)) {
        throw new Error("Can't start parsing from rule \"" + options.startRule + "\".");
      }

      peg$startRuleFunction = peg$startRuleFunctions[options.startRule];
    }

    function text() {
      return input.substring(peg$reportedPos, peg$currPos);
    }

    function offset() {
      return peg$reportedPos;
    }

    function line() {
      return peg$computePosDetails(peg$reportedPos).line;
    }

    function column() {
      return peg$computePosDetails(peg$reportedPos).column;
    }

    function expected(description) {
      throw peg$buildException(null, [{ type: "other", description: description }], peg$reportedPos);
    }

    function error(message) {
      throw peg$buildException(message, null, peg$reportedPos);
    }

    function peg$computePosDetails(pos) {
      function advance(details, startPos, endPos) {
        var p, ch;

        for (p = startPos; p < endPos; p++) {
          ch = input.charAt(p);
          if (ch === "\n") {
            if (!details.seenCR) {
              details.line++;
            }
            details.column = 1;
            details.seenCR = false;
          } else if (ch === "\r" || ch === "\u2028" || ch === "\u2029") {
            details.line++;
            details.column = 1;
            details.seenCR = true;
          } else {
            details.column++;
            details.seenCR = false;
          }
        }
      }

      if (peg$cachedPos !== pos) {
        if (peg$cachedPos > pos) {
          peg$cachedPos = 0;
          peg$cachedPosDetails = { line: 1, column: 1, seenCR: false };
        }
        advance(peg$cachedPosDetails, peg$cachedPos, pos);
        peg$cachedPos = pos;
      }

      return peg$cachedPosDetails;
    }

    function peg$fail(expected) {
      if (peg$currPos < peg$maxFailPos) {
        return;
      }

      if (peg$currPos > peg$maxFailPos) {
        peg$maxFailPos = peg$currPos;
        peg$maxFailExpected = [];
      }

      peg$maxFailExpected.push(expected);
    }

    function peg$buildException(message, expected, pos) {
      function cleanupExpected(expected) {
        var i = 1;

        expected.sort(function (a, b) {
          if (a.description < b.description) {
            return -1;
          } else if (a.description > b.description) {
            return 1;
          } else {
            return 0;
          }
        });

        while (i < expected.length) {
          if (expected[i - 1] === expected[i]) {
            expected.splice(i, 1);
          } else {
            i++;
          }
        }
      }

      function buildMessage(expected, found) {
        function stringEscape(s) {
          function hex(ch) {
            return ch.charCodeAt(0).toString(16).toUpperCase();
          }

          return s.replace(/\\/g, "\\\\").replace(/"/g, "\\\"").replace(/\x08/g, "\\b").replace(/\t/g, "\\t").replace(/\n/g, "\\n").replace(/\f/g, "\\f").replace(/\r/g, "\\r").replace(/[\x00-\x07\x0B\x0E\x0F]/g, function (ch) {
            return "\\x0" + hex(ch);
          }).replace(/[\x10-\x1F\x80-\xFF]/g, function (ch) {
            return "\\x" + hex(ch);
          }).replace(/[\u0180-\u0FFF]/g, function (ch) {
            return "\\u0" + hex(ch);
          }).replace(/[\u1080-\uFFFF]/g, function (ch) {
            return "\\u" + hex(ch);
          });
        }

        var expectedDescs = new Array(expected.length),
            expectedDesc,
            foundDesc,
            i;

        for (i = 0; i < expected.length; i++) {
          expectedDescs[i] = expected[i].description;
        }

        expectedDesc = expected.length > 1 ? expectedDescs.slice(0, -1).join(", ") + " or " + expectedDescs[expected.length - 1] : expectedDescs[0];

        foundDesc = found ? "\"" + stringEscape(found) + "\"" : "end of input";

        return "Expected " + expectedDesc + " but " + foundDesc + " found.";
      }

      var posDetails = peg$computePosDetails(pos),
          found = pos < input.length ? input.charAt(pos) : null;

      if (expected !== null) {
        cleanupExpected(expected);
      }

      return new SyntaxError(message !== null ? message : buildMessage(expected, found), expected, found, pos, posDetails.line, posDetails.column);
    }

    function peg$parsestart() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsecall();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsecall();
      }
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parsews();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parsews();
        }
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c2(s1);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsecall() {
      var s0, s1, s2, s3, s4, s5;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parsecall_exprs();
        if (s2 !== peg$FAILED) {
          s3 = peg$parselb();
          if (s3 !== peg$FAILED) {
            s4 = peg$parseproperty_list();
            if (s4 !== peg$FAILED) {
              s5 = peg$parserb();
              if (s5 !== peg$FAILED) {
                peg$reportedPos = s0;
                s1 = peg$c3(s2, s4);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$c0;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$c0;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$c0;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsecall_exprs() {
      var s0, s1;

      s0 = [];
      s1 = peg$parsecall_expr_parent();
      if (s1 === peg$FAILED) {
        s1 = peg$parsecall_expr();
      }
      while (s1 !== peg$FAILED) {
        s0.push(s1);
        s1 = peg$parsecall_expr_parent();
        if (s1 === peg$FAILED) {
          s1 = peg$parsecall_expr();
        }
      }

      return s0;
    }

    function peg$parsecall_expr_parent() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = peg$parsecall_expr();
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 46) {
          s2 = peg$c4;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) {
            peg$fail(peg$c5);
          }
        }
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c6(s1);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsecall_expr() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      s1 = peg$currPos;
      s2 = [];
      if (peg$c7.test(input.charAt(peg$currPos))) {
        s3 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c8);
        }
      }
      if (s3 !== peg$FAILED) {
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          if (peg$c7.test(input.charAt(peg$currPos))) {
            s3 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) {
              peg$fail(peg$c8);
            }
          }
        }
      } else {
        s2 = peg$c0;
      }
      if (s2 !== peg$FAILED) {
        s2 = input.substring(s1, peg$currPos);
      }
      s1 = s2;
      if (s1 !== peg$FAILED) {
        s2 = peg$parsecall_expr_args();
        if (s2 === peg$FAILED) {
          s2 = peg$c9;
        }
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c10(s1, s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsecall_expr_args() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 40) {
        s1 = peg$c11;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c12);
        }
      }
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parsefirst_arg();
        if (s3 === peg$FAILED) {
          s3 = peg$parsearg();
        }
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parsefirst_arg();
          if (s3 === peg$FAILED) {
            s3 = peg$parsearg();
          }
        }
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 41) {
            s3 = peg$c13;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) {
              peg$fail(peg$c14);
            }
          }
          if (s3 !== peg$FAILED) {
            peg$reportedPos = s0;
            s1 = peg$c15(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$c0;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsefirst_arg() {
      var s0, s1, s2, s3, s4;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$currPos;
        s3 = peg$parsearg();
        if (s3 !== peg$FAILED) {
          s3 = input.substring(s2, peg$currPos);
        }
        s2 = s3;
        if (s2 !== peg$FAILED) {
          s3 = [];
          s4 = peg$parsews();
          while (s4 !== peg$FAILED) {
            s3.push(s4);
            s4 = peg$parsews();
          }
          if (s3 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 44) {
              s4 = peg$c16;
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) {
                peg$fail(peg$c17);
              }
            }
            if (s4 !== peg$FAILED) {
              peg$reportedPos = s0;
              s1 = peg$c18(s2);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$c0;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$c0;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsearg() {
      var s0;

      s0 = peg$parsequoted_text_arg();
      if (s0 === peg$FAILED) {
        s0 = peg$parsetext_arg();
      }

      return s0;
    }

    function peg$parsequoted_text_arg() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 39) {
        s1 = peg$c19;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c20);
        }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parsetext_arg();
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 39) {
            s3 = peg$c19;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) {
              peg$fail(peg$c20);
            }
          }
          if (s3 !== peg$FAILED) {
            peg$reportedPos = s0;
            s1 = peg$c21(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$c0;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsetext_arg() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      s1 = peg$currPos;
      s2 = [];
      if (peg$c22.test(input.charAt(peg$currPos))) {
        s3 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c23);
        }
      }
      if (s3 !== peg$FAILED) {
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          if (peg$c22.test(input.charAt(peg$currPos))) {
            s3 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) {
              peg$fail(peg$c23);
            }
          }
        }
      } else {
        s2 = peg$c0;
      }
      if (s2 !== peg$FAILED) {
        s2 = input.substring(s1, peg$currPos);
      }
      s1 = s2;
      if (s1 !== peg$FAILED) {
        peg$reportedPos = s0;
        s1 = peg$c24(s1);
      }
      s0 = s1;

      return s0;
    }

    function peg$parseproperty_list() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parseproperty_list_item();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parseproperty_list_item();
      }
      if (s1 !== peg$FAILED) {
        peg$reportedPos = s0;
        s1 = peg$c6(s1);
      }
      s0 = s1;

      return s0;
    }

    function peg$parseproperty_list_item() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseproperty();
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c6(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parseproperty() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = peg$parsecall();
      if (s1 === peg$FAILED) {
        s1 = peg$parsename();
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parsecomma();
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c25(s1);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsename() {
      var s0, s1, s2, s3, s4;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$currPos;
        s3 = [];
        if (peg$c26.test(input.charAt(peg$currPos))) {
          s4 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s4 = peg$FAILED;
          if (peg$silentFails === 0) {
            peg$fail(peg$c27);
          }
        }
        if (s4 !== peg$FAILED) {
          while (s4 !== peg$FAILED) {
            s3.push(s4);
            if (peg$c26.test(input.charAt(peg$currPos))) {
              s4 = input.charAt(peg$currPos);
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) {
                peg$fail(peg$c27);
              }
            }
          }
        } else {
          s3 = peg$c0;
        }
        if (s3 !== peg$FAILED) {
          s3 = input.substring(s2, peg$currPos);
        }
        s2 = s3;
        if (s2 !== peg$FAILED) {
          peg$reportedPos = s0;
          s1 = peg$c28(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsecomma() {
      var s0, s1, s2, s3;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s2 = peg$c16;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) {
            peg$fail(peg$c17);
          }
        }
        if (s2 === peg$FAILED) {
          s2 = [];
          s3 = peg$parsews();
          while (s3 !== peg$FAILED) {
            s2.push(s3);
            s3 = peg$parsews();
          }
        }
        if (s2 !== peg$FAILED) {
          s1 = [s1, s2];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parselb() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 123) {
          s2 = peg$c29;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) {
            peg$fail(peg$c30);
          }
        }
        if (s2 !== peg$FAILED) {
          s1 = [s1, s2];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parserb() {
      var s0, s1, s2;

      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parsews();
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parsews();
      }
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 125) {
          s2 = peg$c31;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) {
            peg$fail(peg$c32);
          }
        }
        if (s2 !== peg$FAILED) {
          s1 = [s1, s2];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$c0;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$c0;
      }

      return s0;
    }

    function peg$parsews() {
      var s0;

      if (peg$c33.test(input.charAt(peg$currPos))) {
        s0 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c34);
        }
      }

      return s0;
    }

    function peg$parseinteger() {
      var s0, s1, s2;

      peg$silentFails++;
      s0 = peg$currPos;
      s1 = [];
      if (peg$c36.test(input.charAt(peg$currPos))) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c37);
        }
      }
      if (s2 !== peg$FAILED) {
        while (s2 !== peg$FAILED) {
          s1.push(s2);
          if (peg$c36.test(input.charAt(peg$currPos))) {
            s2 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s2 = peg$FAILED;
            if (peg$silentFails === 0) {
              peg$fail(peg$c37);
            }
          }
        }
      } else {
        s1 = peg$c0;
      }
      if (s1 !== peg$FAILED) {
        peg$reportedPos = s0;
        s1 = peg$c38(s1);
      }
      s0 = s1;
      peg$silentFails--;
      if (s0 === peg$FAILED) {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) {
          peg$fail(peg$c35);
        }
      }

      return s0;
    }

    function deepmerge(target, src) {
      var array = Array.isArray(src);
      var dst = array && [] || {};

      if (array) {
        target = target || [];
        dst = dst.concat(target);
        src.forEach(function (e, i) {
          if (typeof dst[i] === "undefined") {
            dst[i] = e;
          } else if (typeof e === "object") {
            dst[i] = deepmerge(target[i], e);
          } else {
            if (target.indexOf(e) === -1) {
              dst.push(e);
            }
          }
        });
      } else {
        if (target && typeof target === "object") {
          Object.keys(target).forEach(function (key) {
            dst[key] = target[key];
          });
        }
        Object.keys(src).forEach(function (key) {
          if (typeof src[key] !== "object" || !src[key]) {
            dst[key] = src[key];
          } else {
            if (!target[key]) {
              dst[key] = src[key];
            } else {
              dst[key] = deepmerge(target[key], src[key]);
            }
          }
        });
      }

      return dst;
    }
    function flat(props) {
      return props.reduce(function (o, p) {
        if (!o[p.name]) {
          o[p.name] = p;
          return o;
        }
        if (p.kind == "property") {
          if (o[p.name].kind != "property") {
            throw "property " + p.name + " clashes with call of same name";
          }
          return o;
        }
        o[p.name] = deepmerge(o[p.name], p);
        return o;
      }, {});
    }
    function expression(e, props) {
      if (!e[0][0]) {
        throw "invalid name";
      }
      var o = {
        kind: "edge",
        name: e[0][0],
        filters: e.slice(1).reduce(function (o, f) {
          o[f[0]] = f[1];
          return o;
        }, {}),
        props: flat(props),
        args: e[0][1]
      };
      return o;
    }

    peg$result = peg$startRuleFunction();

    if (peg$result !== peg$FAILED && peg$currPos === input.length) {
      return peg$result;
    } else {
      if (peg$result !== peg$FAILED && peg$currPos < input.length) {
        peg$fail({ type: "end", description: "end of input" });
      }

      throw peg$buildException(null, peg$maxFailExpected, peg$maxFailPos);
    }
  }

  return {
    SyntaxError: SyntaxError,
    parse: parse
  };
})();

},{}],2:[function(require,module,exports){
'use strict';

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { 'default': obj }; }

function _toConsumableArray(arr) { if (Array.isArray(arr)) { for (var i = 0, arr2 = Array(arr.length); i < arr.length; i++) arr2[i] = arr[i]; return arr2; } else { return Array.from(arr); } }

var _graphql = require('./graphql');

var _graphql2 = _interopRequireDefault(_graphql);

(function () {

	var listeners = {};
	var tests = [];
	var ddl = [];
	var schema = {};
	var actions = {};

	function action(name, fn) {
		actions[name] = fn;
	}

	function addListener(kind, op, table, fn) {
		table.trim().split(/\s/g).forEach(function (table) {
			listeners[table] = op.trim().split(/\s/g).reduce(function (ops, op) {
				ops[kind + '-' + op].push(fn);
				return ops;
			}, listeners[table] || {
				'before-update': [],
				'after-update': [],
				'before-insert': [],
				'after-insert': [],
				'before-delete': [],
				'after-delete': []
			});
		});
	}

	function before(op, table, fn) {
		addListener('before', op, table, fn);
	}

	function after(op, table, fn) {
		addListener('after', op, table, fn);
	}

	function col() {
		var _ref = arguments[0] === undefined ? {} : arguments[0];

		var _ref$type = _ref.type;
		var type = _ref$type === undefined ? 'text' : _ref$type;
		var _ref$nullable = _ref.nullable;
		var nullable = _ref$nullable === undefined ? false : _ref$nullable;
		var _ref$def = _ref.def;
		var def = _ref$def === undefined ? undefined : _ref$def;
		var _ref$onDelete = _ref.onDelete;
		var onDelete = _ref$onDelete === undefined ? 'CASCADE' : _ref$onDelete;
		var _ref$onUpdate = _ref.onUpdate;
		var onUpdate = _ref$onUpdate === undefined ? 'RESTRICT' : _ref$onUpdate;
		var ref = _ref.ref;

		if (type == 'timestamp') {
			type = 'timestamptz'; // never use non timezone stamp - it's bad news.
		}
		var x = [type];
		if (ref) {
			x.push('REFERENCES ' + plv8.quote_ident(ref));
			x.push('ON DELETE ' + onDelete);
			x.push('ON UPDATE ' + onUpdate);
		}
		if (!nullable) {
			x.push('NOT NULL');
		}
		if (def === undefined) {
			switch (type) {
				case 'boolean':
					def = 'false';break;
				case 'json':
					def = '\'{}\'';break;
				case 'timestampz':
					def = 'now()';break;
				default:
					def = null;break;
			}
		}
		if (def !== null) {
			x.push('DEFAULT ' + def);
		}
		return x.join(' ');
	}

	function define(name, o) {
		if (!o.properties) {
			o.properties = {};
		}
		if (!o.edges) {
			o.edges = {};
		}
		var alter = function alter(stmt) {
			if (name != 'root') {
				ddl.push(stmt);
			}
		};
		o.name = name;
		alter('CREATE TABLE ' + plv8.quote_ident(name) + ' ()');
		alter('CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON ' + plv8.quote_ident(name) + ' FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger(\'before\')');
		alter('CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON ' + plv8.quote_ident(name) + ' DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger(\'after\')');
		for (var k in o.properties) {
			alter('ALTER TABLE ' + plv8.quote_ident(name) + ' ADD COLUMN ' + plv8.quote_ident(k) + ' ' + col(o.properties[k]));
		}
		if (!o.properties.id) {
			alter('ALTER TABLE ' + plv8.quote_ident(name) + ' ADD COLUMN id UUID PRIMARY KEY DEFAULT uuid_generate_v4()');
			o.properties.id = { type: 'uuid' };
		}
		if (o.indexes) {
			for (var k in o.indexes) {
				var idx = o.indexes[k];
				var using = idx.using ? 'USING ' + idx.using : '';
				alter('CREATE ' + (idx.unique ? 'UNIQUE' : '') + ' INDEX ' + name + '_' + k + '_idx ON ' + plv8.quote_ident(name) + ' ' + using + ' ( ' + idx.on.map(function (c) {
					return plv8.quote_ident(c);
				}).join(',') + ' )');
			}
		}
		if (o.beforeChange) {
			before('insert update', name, o.beforeChange);
		}
		if (o.afterChange) {
			after('insert update', name, o.afterChange);
		}
		if (o.beforeUpdate) {
			before('update', name, o.beforeUpdate);
		}
		if (o.afterUpdate) {
			after('update', name, o.afterUpdate);
		}
		if (o.beforeInsert) {
			before('insert', name, o.beforeInsert);
		}
		if (o.afterInsert) {
			after('insert', name, o.afterInsert);
		}
		if (o.beforeDelete) {
			before('delete', name, o.beforeDelete);
		}
		if (o.afterDelete) {
			after('delete', name, o.afterDelete);
		}
		schema[name] = o;
	}

	function defineJoin(tables, o) {
		if (!o) {
			o = {};
		}
		o.properties = Object.assign(o.properties || {}, tables.reduce(function (o, name) {
			o[name + '_id'] = { type: 'uuid', ref: name };
			return o;
		}, {}));
		o.indexes = Object.assign(o.indexes || {}, {
			join: {
				unique: true,
				on: tables.map(function (t) {
					return t + '_id';
				})
			}
		});
		define(tables.join('_'), o);
	}

	var PARENT_MATCH = /\$this/g;

	function gqlToSql(token, _ref2, _ref3, parent) {
		var name = _ref2.name;
		var properties = _ref2.properties;
		var edges = _ref2.edges;
		var args = _ref3.args;
		var props = _ref3.props;
		var filters = _ref3.filters;
		var i = arguments[4] === undefined ? 0 : arguments[4];
		var pl = arguments[5] === undefined ? 0 : arguments[5];

		var cols = Object.keys(props || {}).map(function (k) {
			var o = props[k];
			switch (o.kind) {
				case 'property':
					if (!parent) {
						throw 'the root entity does not have any properties: ' + k + ' is not valid here';
					}
					return parent + '.' + k;
				case 'edge':
					var x = 'x' + i;
					var q = 'q' + i;
					var call = edges[k];
					if (!call) {
						throw 'no such edge ' + k + ' for ' + name;
					}
					var edge = call.apply(token, o.args);
					if (!edge.query) {
						throw 'missing query for edge call ' + k + ' on ' + name;
					}
					var sql = edge.query;
					if (Array.isArray(sql)) {
						sql = sql[0].replace(/\$(\d+)/, function (match, ns) {
							var n = parseInt(ns, 10);
							if (n <= 0) {
								throw 'invalid placeholder name: $' + ns;
							}
							if (sql.length - 1 < n) {
								throw 'no variable for placeholder: $' + ns;
							}
							return plv8.quote_literal(sql[n]);
						});
					}
					if (!sql) {
						throw 'no query sql returned from edge call: ' + k + ' on ' + name;
					}
					// Replace special variables
					sql = sql.replace(PARENT_MATCH, function (match) {
						if (!parent) {
							throw 'Cannot use $this table replacement on root calls';
						}
						return plv8.quote_ident(parent);
					});
					var jsonfn = edge.type == 'array' ? 'json_agg' : 'row_to_json';
					var type = (edge.type == 'array' ? edge.of : edge.type) || 'raw';
					if (type == 'raw') {
						return '\n\t\t\t\t\t\t(with\n\t\t\t\t\t\t\t' + q + ' as ( ' + sql + ' )\n\t\t\t\t\t\t\tselect ' + jsonfn + '(' + q + '.*) from ' + q + '\n\t\t\t\t\t\t) as ' + k + '\n\t\t\t\t\t';
					}
					var table = schema[type];
					if (!table) {
						throw 'unknown return type ' + type + ' for edge call ' + k + ' on ' + name;
					}
					return '\n\t\t\t\t\t(with\n\t\t\t\t\t\t' + q + ' as ( ' + sql + ' ),\n\t\t\t\t\t\t' + x + ' as ( ' + gqlToSql(token, table, o, q, ++i) + ' from ' + q + ' )\n\t\t\t\t\t\tselect ' + jsonfn + '(' + x + '.*) from ' + x + '\n\t\t\t\t\t) as ' + k + '\n\t\t\t\t';
				default:
					throw 'unknown property type: ' + o.kind;
			}
		});
		return 'select ' + cols.join(',');
	}

	arla.trigger = function (e) {
		var op = e.opKind + '-' + e.op;
		['*', e.table].forEach(function (table) {
			var ops = listeners[table];
			if (!ops || ops.length == 0) {
				return;
			}
			var triggers = ops[op];
			if (!triggers) {
				return;
			}
			triggers.forEach(function (fn) {
				try {
					fn(e.record, e);
				} catch (err) {
					console.log(op + ' constraint rejected transaction for ' + e.table + ' record: ' + JSON.stringify(e.record, null, 4));
					throw err;
				}
			});
		});
		return e.record;
	};

	arla.replay = function (m) {
		arla.exec(m.Name, m.Token, m.Args);
		return true;
	};

	arla.exec = function (name, token, args) {
		var _db;

		var fn = actions[name];
		if (!fn) {
			if (/^[a-zA-Z0-9_]+$/.test(name)) {
				throw 'no such action ' + name;
			} else {
				throw 'invalid action';
			}
		}
		// exec the mutation func
		console.debug('action ' + name + ' given', args);
		var queryArgs = fn.apply(token, args);
		if (!queryArgs) {
			console.debug('action ' + name + ' was a noop');
			return [];
		}
		console.debug('action ' + name + ' returned', queryArgs);
		if (!Array.isArray(queryArgs)) {
			queryArgs = [queryArgs];
		}
		// ensure first arg is valid
		if (typeof queryArgs[0] != 'string') {
			throw 'invalid response from action. should be: [sqlstring, ...args]';
		}
		// run the query returned from the mutation func
		return (_db = db).query.apply(_db, _toConsumableArray(queryArgs));
	};

	arla.query = function (token, query) {
		if (!query) {
			throw new SyntaxError('arla_query: query text cannot be null');
		}
		query = 'root(){ ' + query + ' }';
		try {
			console.debug('QUERY:', token, query);
			var ast = _graphql2['default'].parse(query);
			console.debug('AST:', ast);
			var sql = gqlToSql(token, schema.root, ast[0]);
			var res = db.query(sql)[0];
			console.debug('RESULT', res);
			return res;
		} catch (err) {
			if (err.line && err.offset) {
				console.warn(query.split(/\n/)[err.line - 1]);
				console.warn(Array(err.column).join('-') + '^');
				throw new SyntaxError('arla_query: line ' + err.line + ', column ' + err.column + ': ' + err.message);
			}
			throw err;
		}
	};

	arla.authenticate = function (values) {
		var res = db.query.apply(db, arla.cfg.authenticate(values));
		if (res.length < 1) {
			throw new Error('unauthorized');
		}
		return res[0];
	};

	arla.register = function (values) {
		return arla.cfg.register(values);
	};

	arla.init = function () {
		try {
			// build schema
			ddl.forEach(function (stmt) {
				db.query(stmt);
			});
		} catch (e) {
			arla.throwError(e);
		}
	};

	arla.configure = function (cfg) {
		if (arla.cfg) {
			throw 'configure should only be called ONCE!';
		}
		// setup user schema
		Object.keys(cfg.schema || {}).forEach(function (name) {
			define(name, cfg.schema[name]);
		});
		// setup user actions
		var actionNames = Object.keys(cfg.actions || {});
		actionNames.forEach(function (name) {
			action(name, cfg.actions[name]);
		});
		cfg.actions = actionNames;
		// store cfg for later
		arla.cfg = cfg;
		// validate some cfg options
		if (!arla.cfg.authenticate) {
			throw 'missing required "authenticate" function';
		}
		if (!arla.cfg.register) {
			throw 'missing required "register" function';
		}
		// evaluate other config options
		for (var k in arla.cfg) {
			switch (k) {
				case 'schema':
				case 'actions':
				case 'authenticate':
				case 'register':
					break;
				case 'logLevel':
					plv8.elog(NOTICE, 'setting logLevel:' + arla.cfg[k]);
					console.logLevel = arla.cfg[k];
					break;
				default:
					console.warn('ignoring invalid config option:', k);
			}
		}
	};

	arla.throwError = function (e) {
		plv8.elog(ERROR, e.stack || e.message || e.toString());
	};
})();

// Execute the user's code
try {} catch (e) {
	arla.throwError(e);
}

//CONFIG//

},{"./graphql":1}]},{},[2]);

$javascript$ LANGUAGE "plv8";

-- fire normalizes trigger data into an "op" event that is emitted.
CREATE OR REPLACE FUNCTION arla_fire_trigger() RETURNS trigger AS $$
	return plv8.arla.trigger({
		opKind: TG_ARGV[0],
		op: TG_OP.toLowerCase(),
		table: TG_TABLE_NAME.toLowerCase(),
		record: TG_OP=="DELETE" ? OLD : NEW,
		old: TG_OP=="INSERT" ? {} : OLD,
		args: TG_ARGV.slice(1)
	});
$$ LANGUAGE "plv8";


-- execute a mutation
CREATE OR REPLACE FUNCTION arla_exec(name text, t json, args json) RETURNS json AS $$
	try {
		return JSON.stringify(plv8.arla.exec(name, t, args, false));
	} catch (e) {
		plv8.arla.throwError(e);
	}
$$ LANGUAGE "plv8";

-- execute a mutation using a json representation of the mutation
CREATE OR REPLACE FUNCTION arla_replay(mutation json) RETURNS boolean AS $$
	try {
		return plv8.arla.replay(mutation);
	} catch (e) {
		plv8.arla.throwError(e);
	}
$$ LANGUAGE "plv8";

-- use graphql to execute a query
CREATE OR REPLACE FUNCTION arla_query(t json, q text) RETURNS json AS $$
	try {
		return JSON.stringify(plv8.arla.query(t, q));
	} catch (e) {
		plv8.arla.throwError(e);
	}
$$ LANGUAGE "plv8";

-- run the authentication func
CREATE OR REPLACE FUNCTION arla_authenticate(vals json) RETURNS json AS $$
	try {
		return JSON.stringify(plv8.arla.authenticate(vals));
	} catch (e) {
		plv8.arla.throwError(e);
	}
$$ LANGUAGE "plv8";

-- run the registration transformation func
CREATE OR REPLACE FUNCTION arla_register(vals json) RETURNS json AS $$
	try {
		return JSON.stringify(plv8.arla.register(vals));
	} catch (e) {
		plv8.arla.throwError(e);
	}
$$ LANGUAGE "plv8";

-- since(x) === age(now(), x)
CREATE OR REPLACE FUNCTION since(t timestamptz) RETURNS interval AS $$
	select age(now(), t);
$$ LANGUAGE "sql" VOLATILE;

-- until(x) === age(x, now())
CREATE OR REPLACE FUNCTION until(t timestamptz) RETURNS interval AS $$
	select age(t, now());
$$ LANGUAGE "sql" VOLATILE;

DO $$
  plv8.arla.init();
$$ LANGUAGE plv8;
`
