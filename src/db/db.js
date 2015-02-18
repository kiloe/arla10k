import * as console from "./console";

export function query(sql, ...args){
	console.debug("QUERY", sql);
	if( args.length > 0 ){
		console.debug("ARGS:", ...args);
	}
	return plv8.execute(sql, ...args);
}

export function transaction(fn){
	plv8.subtransaction(function(){
		fn();
	});
}

export function insert(table, o){
	var sql = `insert into ${table} ( ${ Object.keys(o).join(',') } )
				values ( ${ Object.keys(o).map( (_, i) => '$'+(i+1)) } )
				returning *`;
	var r = plv8.execute(sql, Object.keys(o).map( k => o[k] ))[0];
	console.debug(`created new ${table} record: ${ JSON.stringify(r, null, 4) }`);
	return r;
}

export function count(sql, ...args){
	return query(`WITH x AS (${sql}) SELECT count(*) AS c FROM x`, ...args)[0].c;
}

export default {
	query,
	transaction,
	insert,
	count
}
