import {define, action, sql_exports} from "./runtime";

// register tables from app config
import * as schema from "./app/schema";
Object.keys(schema).forEach(function(k){
	define(k, schema[k]);
});
// register actions from app config
import * as actions from "./app/actions";
Object.keys(actions).forEach(function(k){
	action(k, actions[k]);
});
// attach all runtime functions to plv8 global
plv8.functions = sql_exports;
