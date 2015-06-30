import {define, action, sql_exports} from "./runtime";

arla.configure = function(cfg){
	define('meta', {
	properties: {
		key:   {type: 'text', unique: true},
		value: {type:'text'}
	}
	});

	Object.keys(cfg.schema).forEach(function(k){
		define(k, cfg.schema[k]);
	});
	// register actions from app config
	Object.keys(cfg.actions).forEach(function(k){
		action(k, cfg.actions[k]);
	});
	// attach all runtime functions to plv8 global
	plv8.functions = sql_exports;
};
