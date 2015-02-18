import {define, action, sql_exports} from "./runtime";
import * as actions from "./app/actions";
import * as schema from "./app/schema";
import {log} from "./console";

log(schema);

// register tables from app config
Object.keys(schema).forEach(function(k){
	define(k, schema[k]);
});
// register actions from app config
Object.keys(actions).forEach(function(k){
	action(k, actions[k]);
});
// attach all runtime functions to plv8 global
plv8.functions = sql_exports;
