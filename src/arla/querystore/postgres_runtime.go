package querystore
const postgresRuntimeScript = `
(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
// http://wiki.commonjs.org/wiki/Unit_Testing/1.0
//
// THIS IS NOT TESTED NOR LIKELY TO WORK OUTSIDE V8!
//
// Copyright (c) 2011 Jxck
//
// Originally from node.js (http://nodejs.org)
// Copyright Joyent, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the 'Software'), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

'use strict';

module.exports = (function () {

  // UTILITY

  // Object.create compatible in IE
  var create = Object.create || function (p) {
    if (!p) throw Error('no type');
    function f() {};
    f.prototype = p;
    return new f();
  };

  // UTILITY
  var util = {
    inherits: function inherits(ctor, superCtor) {
      ctor.super_ = superCtor;
      ctor.prototype = create(superCtor.prototype, {
        constructor: {
          value: ctor,
          enumerable: false,
          writable: true,
          configurable: true
        }
      });
    },
    isArray: function isArray(ar) {
      return Array.isArray(ar);
    },
    isBoolean: function isBoolean(arg) {
      return typeof arg === 'boolean';
    },
    isNull: function isNull(arg) {
      return arg === null;
    },
    isNullOrUndefined: function isNullOrUndefined(arg) {
      return arg == null;
    },
    isNumber: function isNumber(arg) {
      return typeof arg === 'number';
    },
    isString: function isString(arg) {
      return typeof arg === 'string';
    },
    isSymbol: function isSymbol(arg) {
      return typeof arg === 'symbol';
    },
    isUndefined: function isUndefined(arg) {
      return arg === void 0;
    },
    isRegExp: function isRegExp(re) {
      return util.isObject(re) && util.objectToString(re) === '[object RegExp]';
    },
    isObject: function isObject(arg) {
      return typeof arg === 'object' && arg !== null;
    },
    isDate: function isDate(d) {
      return util.isObject(d) && util.objectToString(d) === '[object Date]';
    },
    isError: function isError(e) {
      return isObject(e) && (objectToString(e) === '[object Error]' || e instanceof Error);
    },
    isFunction: function isFunction(arg) {
      return typeof arg === 'function';
    },
    isPrimitive: function isPrimitive(arg) {
      return arg === null || typeof arg === 'boolean' || typeof arg === 'number' || typeof arg === 'string' || typeof arg === 'symbol' || // ES6 symbol
      typeof arg === 'undefined';
    },
    objectToString: function objectToString(o) {
      return Object.prototype.toString.call(o);
    }
  };

  var pSlice = Array.prototype.slice;

  // From https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/keys
  var Object_keys = typeof Object.keys === 'function' ? Object.keys : (function () {
    var hasOwnProperty = Object.prototype.hasOwnProperty,
        hasDontEnumBug = !({ toString: null }).propertyIsEnumerable('toString'),
        dontEnums = ['toString', 'toLocaleString', 'valueOf', 'hasOwnProperty', 'isPrototypeOf', 'propertyIsEnumerable', 'constructor'],
        dontEnumsLength = dontEnums.length;

    return function (obj) {
      if (typeof obj !== 'object' && (typeof obj !== 'function' || obj === null)) {
        throw new TypeError('Object.keys called on non-object');
      }

      var result = [],
          prop,
          i;

      for (prop in obj) {
        if (hasOwnProperty.call(obj, prop)) {
          result.push(prop);
        }
      }

      if (hasDontEnumBug) {
        for (i = 0; i < dontEnumsLength; i++) {
          if (hasOwnProperty.call(obj, dontEnums[i])) {
            result.push(dontEnums[i]);
          }
        }
      }
      return result;
    };
  })();

  // 1. The assert module provides functions that throw
  // AssertionError's when particular conditions are not met. The
  // assert module must conform to the following interface.

  var assert = ok;

  // 2. The AssertionError is defined in assert.
  // new assert.AssertionError({ message: message,
  //                             actual: actual,
  //                             expected: expected })

  assert.AssertionError = function AssertionError(options) {
    this.name = 'AssertionError';
    this.actual = options.actual;
    this.expected = options.expected;
    this.operator = options.operator;
    if (options.message) {
      this.message = options.message;
      this.generatedMessage = false;
    } else {
      this.message = getMessage(this);
      this.generatedMessage = true;
    }
    var stackStartFunction = options.stackStartFunction || fail;
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, stackStartFunction);
    } else {
      // try to throw an error now, and from the stack property
      // work out the line that called in to assert.js.
      try {
        this.stack = new Error().stack.toString();
      } catch (e) {}
    }
  };

  // assert.AssertionError instanceof Error
  util.inherits(assert.AssertionError, Error);

  function replacer(key, value) {
    if (util.isUndefined(value)) {
      return '' + value;
    }
    if (util.isNumber(value) && (isNaN(value) || !isFinite(value))) {
      return value.toString();
    }
    if (util.isFunction(value) || util.isRegExp(value)) {
      return value.toString();
    }
    return value;
  }

  function truncate(s, n) {
    if (util.isString(s)) {
      return s.length < n ? s : s.slice(0, n);
    } else {
      return s;
    }
  }

  function getMessage(self) {
    return truncate(JSON.stringify(self.actual, replacer), 128) + ' ' + self.operator + ' ' + truncate(JSON.stringify(self.expected, replacer), 128);
  }

  // At present only the three keys mentioned above are used and
  // understood by the spec. Implementations or sub modules can pass
  // other keys to the AssertionError's constructor - they will be
  // ignored.

  // 3. All of the following functions must throw an AssertionError
  // when a corresponding condition is not met, with a message that
  // may be undefined if not provided.  All assertion methods provide
  // both the actual and expected values to the assertion error for
  // display purposes.

  function fail(actual, expected, message, operator, stackStartFunction) {
    throw new assert.AssertionError({
      message: message,
      actual: actual,
      expected: expected,
      operator: operator,
      stackStartFunction: stackStartFunction
    });
  }

  // EXTENSION! allows for well behaved errors defined elsewhere.
  assert.fail = fail;

  // 4. Pure assertion tests whether a value is truthy, as determined
  // by !!guard.
  // assert.ok(guard, message_opt);
  // This statement is equivalent to assert.equal(true, !!guard,
  // message_opt);. To test strictly for the value true, use
  // assert.strictEqual(true, guard, message_opt);.

  function ok(value, message) {
    if (!value) fail(value, true, message, '==', assert.ok);
  }
  assert.ok = ok;

  // 5. The equality assertion tests shallow, coercive equality with
  // ==.
  // assert.equal(actual, expected, message_opt);

  assert.equal = function equal(actual, expected, message) {
    if (actual != expected) fail(actual, expected, message, '==', assert.equal);
  };

  // 6. The non-equality assertion tests for whether two objects are not equal
  // with != assert.notEqual(actual, expected, message_opt);

  assert.notEqual = function notEqual(actual, expected, message) {
    if (actual == expected) {
      fail(actual, expected, message, '!=', assert.notEqual);
    }
  };

  // 7. The equivalence assertion tests a deep equality relation.
  // assert.deepEqual(actual, expected, message_opt);

  assert.deepEqual = function deepEqual(actual, expected, message) {
    if (!_deepEqual(actual, expected)) {
      fail(actual, expected, message, 'deepEqual', assert.deepEqual);
    }
  };

  function _deepEqual(actual, expected) {
    // 7.1. All identical values are equivalent, as determined by ===.
    if (actual === expected) {
      return true;

      //  } else if (util.isBuffer(actual) && util.isBuffer(expected)) {
      //    if (actual.length != expected.length) return false;
      //
      //    for (var i = 0; i < actual.length; i++) {
      //      if (actual[i] !== expected[i]) return false;
      //    }
      //
      //    return true;
      //
      // 7.2. If the expected value is a Date object, the actual value is
      // equivalent if it is also a Date object that refers to the same time.
    } else if (util.isDate(actual) && util.isDate(expected)) {
      return actual.getTime() === expected.getTime();

      // 7.3 If the expected value is a RegExp object, the actual value is
      // equivalent if it is also a RegExp object with the same source and
      // properties ('global', 'multiline', 'lastIndex', 'ignoreCase').
    } else if (util.isRegExp(actual) && util.isRegExp(expected)) {
      return actual.source === expected.source && actual.global === expected.global && actual.multiline === expected.multiline && actual.lastIndex === expected.lastIndex && actual.ignoreCase === expected.ignoreCase;

      // 7.4. Other pairs that do not both pass typeof value == 'object',
      // equivalence is determined by ==.
    } else if (!util.isObject(actual) && !util.isObject(expected)) {
      return actual == expected;

      // 7.5 For all other Object pairs, including Array objects, equivalence is
      // determined by having the same number of owned properties (as verified
      // with Object.prototype.hasOwnProperty.call), the same set of keys
      // (although not necessarily the same order), equivalent values for every
      // corresponding key, and an identical 'prototype' property. Note: this
      // accounts for both named and indexed properties on Arrays.
    } else {
      return objEquiv(actual, expected);
    }
  }

  var isArguments = function isArguments(object) {
    return Object.prototype.toString.call(object) == '[object Arguments]';
  };

  (function () {
    if (!isArguments(arguments)) {
      isArguments = function (object) {
        return object != null && typeof object === 'object' && typeof object.callee === 'function' && typeof object.length === 'number' || false;
      };
    }
  })();

  function objEquiv(a, b) {
    if (util.isNullOrUndefined(a) || util.isNullOrUndefined(b)) return false;
    // an identical 'prototype' property.
    if (a.prototype !== b.prototype) return false;
    //~~~I've managed to break Object.keys through screwy arguments passing.
    //   Converting to array solves the problem.
    var aIsArgs = isArguments(a),
        bIsArgs = isArguments(b);
    if (aIsArgs && !bIsArgs || !aIsArgs && bIsArgs) return false;
    if (aIsArgs) {
      a = pSlice.call(a);
      b = pSlice.call(b);
      return _deepEqual(a, b);
    }
    try {
      var ka = Object_keys(a),
          kb = Object_keys(b),
          key,
          i;
    } catch (e) {
      //happens when one is a string literal and the other isn't
      return false;
    }
    // having the same number of owned properties (keys incorporates
    // hasOwnProperty)
    if (ka.length != kb.length) return false;
    //the same set of keys (although not necessarily the same order),
    ka.sort();
    kb.sort();
    //~~~cheap key test
    for (i = ka.length - 1; i >= 0; i--) {
      if (ka[i] != kb[i]) return false;
    }
    //equivalent values for every corresponding key, and
    //~~~possibly expensive deep test
    for (i = ka.length - 1; i >= 0; i--) {
      key = ka[i];
      if (!_deepEqual(a[key], b[key])) return false;
    }
    return true;
  }

  // 8. The non-equivalence assertion tests for any deep inequality.
  // assert.notDeepEqual(actual, expected, message_opt);

  assert.notDeepEqual = function notDeepEqual(actual, expected, message) {
    if (_deepEqual(actual, expected)) {
      fail(actual, expected, message, 'notDeepEqual', assert.notDeepEqual);
    }
  };

  // 9. The strict equality assertion tests strict equality, as determined by ===.
  // assert.strictEqual(actual, expected, message_opt);

  assert.strictEqual = function strictEqual(actual, expected, message) {
    if (actual !== expected) {
      fail(actual, expected, message, '===', assert.strictEqual);
    }
  };

  // 10. The strict non-equality assertion tests for strict inequality, as
  // determined by !==.  assert.notStrictEqual(actual, expected, message_opt);

  assert.notStrictEqual = function notStrictEqual(actual, expected, message) {
    if (actual === expected) {
      fail(actual, expected, message, '!==', assert.notStrictEqual);
    }
  };

  function expectedException(actual, expected) {
    if (!actual || !expected) {
      return false;
    }

    if (Object.prototype.toString.call(expected) == '[object RegExp]') {
      return expected.test(actual);
    } else if (actual instanceof expected) {
      return true;
    } else if (expected.call({}, actual) === true) {
      return true;
    }

    return false;
  }

  function _throws(shouldThrow, block, expected, message) {
    var actual;

    if (util.isString(expected)) {
      message = expected;
      expected = null;
    }

    try {
      block();
    } catch (e) {
      actual = e;
    }

    message = (expected && expected.name ? ' (' + expected.name + ').' : '.') + (message ? ' ' + message : '.');

    if (shouldThrow && !actual) {
      fail(actual, expected, 'Missing expected exception' + message);
    }

    if (!shouldThrow && expectedException(actual, expected)) {
      fail(actual, expected, 'Got unwanted exception' + message);
    }

    if (shouldThrow && actual && expected && !expectedException(actual, expected) || !shouldThrow && actual) {
      throw actual;
    }
  }

  // 11. Expected to throw an error:
  // assert.throws(block, Error_opt, message_opt);

  assert.throws = function (block, /*optional*/error, /*optional*/message) {
    _throws.apply(this, [true].concat(pSlice.call(arguments)));
  };

  // EXTENSION! This is annoying to write outside this module.
  assert.doesNotThrow = function (block, /*optional*/message) {
    _throws.apply(this, [false].concat(pSlice.call(arguments)));
  };

  assert.ifError = function (err) {
    if (err) {
      throw err;
    }
  };

  return assert;
})();

},{}],2:[function(require,module,exports){
'use strict';

Object.defineProperty(exports, '__esModule', {
	value: true
});
exports.log = log;
exports.debug = debug;
exports.warn = warn;
exports.error = error;

var SHOW_DEBUG = true;

function logger(level) {
	for (var _len = arguments.length, msgs = Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
		msgs[_key - 1] = arguments[_key];
	}

	var msg = msgs.map(function (msg) {
		if (typeof msg == 'object') {
			msg = JSON.stringify(msg, null, 4);
		}
		return msg;
	}).join(' ');
	(msg || '').split(/\n/g).forEach(function (line) {
		plv8.elog(level, line);
	});
}

function log() {
	for (var _len2 = arguments.length, msg = Array(_len2), _key2 = 0; _key2 < _len2; _key2++) {
		msg[_key2] = arguments[_key2];
	}

	logger.apply(undefined, [NOTICE].concat(msg));
}

function debug() {
	if (SHOW_DEBUG) {
		log.apply(undefined, arguments);
	}
}

function warn() {
	for (var _len3 = arguments.length, msg = Array(_len3), _key3 = 0; _key3 < _len3; _key3++) {
		msg[_key3] = arguments[_key3];
	}

	logger.apply(undefined, [WARNING].concat(msg));
}

function error() {
	for (var _len4 = arguments.length, msg = Array(_len4), _key4 = 0; _key4 < _len4; _key4++) {
		msg[_key4] = arguments[_key4];
	}

	// don't use ERROR as this will halt execution
	logger.apply(undefined, [WARNING].concat(msg));
}

exports['default'] = {
	log: log,
	debug: debug,
	warn: warn,
	error: error
};

},{}],3:[function(require,module,exports){
"use strict";

Object.defineProperty(exports, "__esModule", {
	value: true
});
exports.query = query;
exports.transaction = transaction;
exports.insert = insert;
exports.update = update;
exports.destroy = destroy;
exports.count = count;

function _interopRequireWildcard(obj) { if (obj && obj.__esModule) { return obj; } else { var newObj = {}; if (obj != null) { for (var key in obj) { if (Object.prototype.hasOwnProperty.call(obj, key)) newObj[key] = obj[key]; } } newObj["default"] = obj; return newObj; } }

var _console = require("./console");

var console = _interopRequireWildcard(_console);

function query(sql) {
	var _plv8;

	console.debug("QUERY", sql);

	for (var _len = arguments.length, args = Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
		args[_key - 1] = arguments[_key];
	}

	if (args.length > 0) {
		console.debug.apply(console, ["ARGS:"].concat(args));
	}
	return (_plv8 = plv8).execute.apply(_plv8, [sql].concat(args));
}

function transaction(fn) {
	plv8.subtransaction(function () {
		fn();
	});
}

// Helper for simple inserts:
// db.insert('mytable', {col1:'hello', col2:true});
// db.insert('mytable', {col1:'hello', col2:true}, 'col1 IS NULL');
// db.insert('mytable', {col1:'hello', col2:true}, 'id = $1 AND name = $2', id, name);
// ... for anything more complex, use db.query()

function insert(table, o, condition) {
	for (var _len2 = arguments.length, args = Array(_len2 > 3 ? _len2 - 3 : 0), _key2 = 3; _key2 < _len2; _key2++) {
		args[_key2 - 3] = arguments[_key2];
	}

	var sql = "\n\t\tinsert into " + table + " ( " + Object.keys(o).join(",") + " )\n\t\tvalues ( " + Object.keys(o).map(function (_, i) {
		return "$" + (i + 1 + args.length);
	}) + " )\n\t";
	if (condition) {
		sql += " where " + condition + " ";
	}
	sql += "returning *";
	return plv8.execute(sql, args.concat(Object.keys(o).map(function (k) {
		return o[k];
	})));
}

// Helper for simple updates:
// db.update('mytable', {col1:'hello', col2:true});
// db.update('mytable', {col1:'hello', col2:true}, 'col1 IS NULL');
// db.update('mytable', {col1:'hello', col2:true}, 'id = $1 AND name = $2', id, name);
// ... for anything more complex, use db.query()

function update(table, o, condition) {
	for (var _len3 = arguments.length, args = Array(_len3 > 3 ? _len3 - 3 : 0), _key3 = 3; _key3 < _len3; _key3++) {
		args[_key3 - 3] = arguments[_key3];
	}

	var keyvals = Object.keys(o).map(function (k, i) {
		return [k, "$" + (i + 1 + args.length)].join(" = ");
	}).join(", ");
	var values = Object.keys(o).map(function (k) {
		return o[k];
	});
	var sql = "\n\t\tupdate " + table + "\n\t\tset " + keyvals + "\n\t";
	if (condition) {
		sql += " where " + condition + " ";
	}
	sql += "returning *";
	return plv8.execute(sql, args.concat(values));
}

// Helper for simple deletes:
// db.destroy('mytable');
// db.destroy('mytable', 'col1 IS NULL');
// db.destroy('mytable', 'id = $1', id);
// ... for anything more complex, use db.query()

function destroy(table, condition) {
	var sql = "\n\t\tdelete from " + table + "\n\t";
	if (condition) {
		sql += " where " + condition + " ";
	}
	sql += "returning *";

	for (var _len4 = arguments.length, args = Array(_len4 > 2 ? _len4 - 2 : 0), _key4 = 2; _key4 < _len4; _key4++) {
		args[_key4 - 2] = arguments[_key4];
	}

	return plv8.execute(sql, args);
}

function count(sql) {
	for (var _len5 = arguments.length, args = Array(_len5 > 1 ? _len5 - 1 : 0), _key5 = 1; _key5 < _len5; _key5++) {
		args[_key5 - 1] = arguments[_key5];
	}

	return query.apply(undefined, ["WITH x AS (" + sql + ") SELECT count(*) AS c FROM x"].concat(args))[0].c;
}

exports["default"] = {
	query: query,
	transaction: transaction,
	insert: insert,
	update: update,
	destroy: destroy,
	count: count
};

},{"./console":2}],4:[function(require,module,exports){
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

},{}],5:[function(require,module,exports){
'use strict';

var _runtime = require('./runtime');

arla.configure = function (cfg) {
	(0, _runtime.define)('meta', {
		properties: {
			key: { type: 'text', unique: true },
			value: { type: 'text' }
		}
	});

	Object.keys(cfg.schema).forEach(function (k) {
		(0, _runtime.define)(k, cfg.schema[k]);
	});
	// register actions from app config
	Object.keys(cfg.actions).forEach(function (k) {
		(0, _runtime.action)(k, cfg.actions[k]);
	});
	// attach all runtime functions to plv8 global
	plv8.functions = _runtime.sql_exports;
};

},{"./runtime":7}],6:[function(require,module,exports){
// Object.assign polyfill waiting for harmony
'use strict';

if (!Object.assign) {
	Object.defineProperty(Object, 'assign', {
		enumerable: false,
		configurable: true,
		writable: true,
		value: function value(target, firstSource) {
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

},{}],7:[function(require,module,exports){
"use strict";

Object.defineProperty(exports, "__esModule", {
	value: true
});
exports.action = action;
exports.before = before;
exports.after = after;
exports.define = define;
exports.defineJoin = defineJoin;

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { "default": obj }; }

function _interopRequireWildcard(obj) { if (obj && obj.__esModule) { return obj; } else { var newObj = {}; if (obj != null) { for (var key in obj) { if (Object.prototype.hasOwnProperty.call(obj, key)) newObj[key] = obj[key]; } } newObj["default"] = obj; return newObj; } }

function _toConsumableArray(arr) { if (Array.isArray(arr)) { for (var i = 0, arr2 = Array(arr.length); i < arr.length; i++) arr2[i] = arr[i]; return arr2; } else { return Array.from(arr); } }

require("./polyfill");

var _assert = require("./assert");

var assert = _interopRequireWildcard(_assert);

var _console = require("./console");

var _console2 = _interopRequireDefault(_console);

var _db = require("./db");

var _db2 = _interopRequireDefault(_db);

var _graphql = require("./graphql");

var _graphql2 = _interopRequireDefault(_graphql);

var SHOW_DEBUG = true;

var listeners = {};
var tests = [];
var schema = {};
var ddl = [];
var actions = {};

function action(name, fn) {
	actions[name] = fn.bind(_db2["default"]);
}

function addListener(kind, op, table, fn) {
	table.trim().split(/\s/g).forEach(function (table) {
		listeners[table] = op.trim().split(/\s/g).reduce(function (ops, op) {
			ops[kind + "-" + op].push(fn);
			return ops;
		}, listeners[table] || {
			"before-update": [],
			"after-update": [],
			"before-insert": [],
			"after-insert": [],
			"before-delete": [],
			"after-delete": []
		});
	});
}

function before(op, table, fn) {
	addListener("before", op, table, fn);
}

function after(op, table, fn) {
	addListener("after", op, table, fn);
}

function col() {
	var _ref = arguments[0] === undefined ? {} : arguments[0];

	var _ref$type = _ref.type;
	var type = _ref$type === undefined ? "text" : _ref$type;
	var _ref$nullable = _ref.nullable;
	var nullable = _ref$nullable === undefined ? false : _ref$nullable;
	var _ref$def = _ref.def;
	var def = _ref$def === undefined ? undefined : _ref$def;
	var _ref$onDelete = _ref.onDelete;
	var onDelete = _ref$onDelete === undefined ? "CASCADE" : _ref$onDelete;
	var _ref$onUpdate = _ref.onUpdate;
	var onUpdate = _ref$onUpdate === undefined ? "RESTRICT" : _ref$onUpdate;
	var ref = _ref.ref;

	if (type == "timestamp") {
		type = "timestamptz"; // never use non timezone stamp - it's bad news.
	}
	var x = [type];
	if (ref) {
		x.push("REFERENCES " + plv8.quote_ident(ref));
		x.push("ON DELETE " + onDelete);
		x.push("ON UPDATE " + onUpdate);
	}
	if (!nullable) {
		x.push("NOT NULL");
	}
	if (def === undefined) {
		switch (type) {
			case "boolean":
				def = "false";break;
			case "json":
				def = "'{}'";break;
			case "timestampz":
				def = "now()";break;
			default:
				def = null;break;
		}
	}
	if (def !== null) {
		x.push("DEFAULT " + def);
	}
	return x.join(" ");
}

function define(name, o) {
	if (!o.properties) {
		o.properties = {};
	}
	if (!o.edges) {
		o.edges = {};
	}
	var alter = function alter(s) {
		if (name != "root") {
			ddl.push(s);
		}
	};
	o.name = name;
	alter("CREATE TABLE " + plv8.quote_ident(name) + " ()");
	alter("CREATE TRIGGER before_trigger BEFORE INSERT OR UPDATE OR DELETE ON " + plv8.quote_ident(name) + " FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('before')");
	alter("CREATE CONSTRAINT TRIGGER after_trigger AFTER INSERT OR UPDATE OR DELETE ON " + plv8.quote_ident(name) + " DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE arla_fire_trigger('after')");
	for (var k in o.properties) {
		alter("ALTER TABLE " + plv8.quote_ident(name) + " ADD COLUMN " + plv8.quote_ident(k) + " " + col(o.properties[k]));
	}
	if (!o.properties.id) {
		alter("ALTER TABLE " + plv8.quote_ident(name) + " ADD COLUMN id UUID PRIMARY KEY DEFAULT uuid_generate_v4()");
		o.properties.id = { type: "uuid" };
	}
	if (o.indexes) {
		for (var k in o.indexes) {
			var idx = o.indexes[k];
			var using = idx.using ? "USING " + idx.using : "";
			alter("CREATE " + (idx.unique ? "UNIQUE" : "") + " INDEX " + name + "_" + k + "_idx ON " + plv8.quote_ident(name) + " " + using + " ( " + idx.on.map(function (c) {
				return plv8.quote_ident(c);
			}).join(",") + " )");
		}
	}
	if (o.beforeChange) {
		before("insert update", name, o.beforeChange);
	}
	if (o.afterChange) {
		after("insert update", name, o.afterChange);
	}
	if (o.beforeUpdate) {
		before("update", name, o.beforeUpdate);
	}
	if (o.afterUpdate) {
		after("update", name, o.afterUpdate);
	}
	if (o.beforeInsert) {
		before("insert", name, o.beforeInsert);
	}
	if (o.afterInsert) {
		after("insert", name, o.afterInsert);
	}
	if (o.beforeDelete) {
		before("delete", name, o.beforeDelete);
	}
	if (o.afterDelete) {
		after("delete", name, o.afterDelete);
	}
	schema[name] = o;
}

function defineJoin(tables, o) {
	if (!o) {
		o = {};
	}
	o.properties = Object.assign(o.properties || {}, tables.reduce(function (o, name) {
		o[name + "_id"] = { type: "uuid", ref: name };
		return o;
	}, {}));
	o.indexes = Object.assign(o.indexes || {}, {
		join: {
			unique: true,
			on: tables.map(function (t) {
				return t + "_id";
			})
		}
	});
	define(tables.join("_"), o);
}

var PARENT_MATCH = /\$this/g;
var VIEWER_MATCH = /\$identity/g;

function gqlToSql(viewer, _ref2, _ref3, parent) {
	var name = _ref2.name;
	var properties = _ref2.properties;
	var edges = _ref2.edges;
	var args = _ref3.args;
	var props = _ref3.props;
	var filters = _ref3.filters;
	var i = arguments[4] === undefined ? 0 : arguments[4];

	var cols = Object.keys(props || {}).map(function (k) {
		var o = props[k];
		switch (o.kind) {
			case "property":
				if (!parent) {
					throw "the root entity does not have any properties: " + k + " is not valid here";
				}
				return parent + "." + k;
			case "edge":
				var x = "x" + i;
				var q = "q" + i;
				var call = edges[k];
				if (!call) {
					throw "no such edge " + k + " for " + name;
				}
				var edge = call.apply(undefined, _toConsumableArray(o.args));
				if (!edge.query) {
					throw "missing query for edge call " + k + " on " + name;
				}
				var sql = edge.query;
				// Replace special variables
				sql = sql.replace(VIEWER_MATCH, plv8.quote_literal(viewer));
				sql = sql.replace(PARENT_MATCH, function (match) {
					if (!parent) {
						throw "Cannot use $this table replacement on root calls";
					}
					return plv8.quote_ident(parent);
				});
				var jsonfn = edge.type == "array" ? "json_agg" : "row_to_json";
				var type = (edge.type == "array" ? edge.of : edge.type) || "raw";
				if (type == "raw") {
					return "\n\t\t\t\t\t(with\n\t\t\t\t\t\t" + q + " as ( " + sql + " )\n\t\t\t\t\t\tselect " + jsonfn + "(" + q + ".*) from " + q + "\n\t\t\t\t\t) as " + k + "\n\t\t\t\t";
				}
				var table = schema[type];
				if (!table) {
					throw "unknown return type " + type + " for edge call " + k + " on " + name;
				}
				return "\n\t\t\t\t(with\n\t\t\t\t\t" + q + " as ( " + sql + " ),\n\t\t\t\t\t" + x + " as ( " + gqlToSql(viewer, table, o, q, ++i) + " from " + q + " )\n\t\t\t\t\tselect " + jsonfn + "(" + x + ".*) from " + x + "\n\t\t\t\t) as " + k + "\n\t\t\t";
			default:
				throw "unknown property type: " + o.kind;
		}
	});
	return "select " + cols.join(",");
}

// Functions that will be exposed via SQL
var sql_exports = {
	arla_fire_trigger: function arla_fire_trigger(e) {
		var op = e.opKind + "-" + e.op;
		["*", e.table].forEach(function (table) {
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
					_console2["default"].debug(op + " constraint rejected transaction for " + e.table + " record: " + JSON.stringify(e.record, null, 4));
					throw err;
				}
			});
		});
		return e.record;
	},
	arla_migrate: function arla_migrate() {
		_db2["default"].transaction(function () {
			ddl.forEach(function (stmt) {
				try {
					_db2["default"].query(stmt);
				} catch (err) {
					if (/already exists/i.test(err, toString())) {} else {
						throw err;
					}
				}
			});
		});
	},
	arla_destroy_data: function arla_destroy_data() {
		for (var _name in schema) {
			if (_name == "root") {
				continue;
			}
			_db2["default"].query("delete from " + plv8.quote_ident(_name));
		}
		return true;
	},
	arla_replay: function arla_replay(mutation) {
		sql_exports.arla_exec(mutation.ID, mutation.Name, mutation.Args, true);
		return true;
	},
	arla_exec: function arla_exec(viewer, name, args, replay) {
		if (name == "resolver") {
			throw "no such action resolver"; // HACK
		}
		var fn = actions[name];
		if (!fn) {
			if (/^[a-zA-Z0-9_]+$/.test(name)) {
				throw "no such action " + name;
			} else {
				throw "invalid action";
			}
		}
		try {
			_console2["default"].log("action " + name + " given " + JSON.stringify(args));
			var queryArgs = fn.apply(undefined, _toConsumableArray(args));
			if (!queryArgs) {
				_console2["default"].debug("action " + name + " was a noop");
				return [];
			}
			_console2["default"].log("action " + name + " returned " + JSON.stringify(queryArgs));
			if (!Array.isArray(queryArgs)) {
				queryArgs = [queryArgs];
			}
			// ensure first arg is valid
			if (typeof queryArgs[0] != "string") {
				throw "invalid response from action. should be: [sqlstring, ...args]";
			}
			// replace magic $viewer variable
			queryArgs[0] = queryArgs[0].replace(VIEWER_MATCH, plv8.quote_literal(viewer));
			// run
			return _db2["default"].query.apply(_db2["default"], _toConsumableArray(queryArgs));
		} catch (err) {
			if (!replay) {
				throw err;
			}
			if (!actions.resolver) {
				_console2["default"].debug("There is no 'resolver' function declared");
				throw err;
			}
			var res = actions.resolver.bind(_db2["default"])(err, name, args, actions);
			_console2["default"].debug("action", name, args, "initially failed, but was resolved");
			return res;
		}
	},
	arla_query: function arla_query(viewer, query) {
		query = "root(){ " + query + " }";
		try {
			_console2["default"].debug("QUERY:", viewer, query);
			var ast = _graphql2["default"].parse(query);
			//console.debug("AST:", ast);
			var sql = gqlToSql(viewer, schema.root, ast[0]);
			//console.debug('SQL:', sql);
			var res = _db2["default"].query(sql)[0];
			_console2["default"].debug("RESULT", res);
			return res;
		} catch (err) {
			if (err.line && err.offset) {
				_console2["default"].warn(query.split(/\n/)[err.line - 1]);
				_console2["default"].warn(Array(err.column).join("-") + "^");
				throw new SyntaxError("arla_query: line " + err.line + ", column " + err.column + ": " + err.message);
			}
			throw err;
		}
	}
};
exports.sql_exports = sql_exports;

// ignore ALTER TABLE errors when column exists

},{"./assert":1,"./console":2,"./db":3,"./graphql":4,"./polyfill":6}]},{},[5]);
`
