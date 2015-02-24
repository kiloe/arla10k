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

// Helper for simple inserts:
// db.insert('mytable', {col1:'hello', col2:true});
// db.insert('mytable', {col1:'hello', col2:true}, `col1 IS NULL`);
// db.insert('mytable', {col1:'hello', col2:true}, `id = $1 AND name = $2`, id, name);
// ... for anything more complex, use db.query()
export function insert(table, o, condition, ...args){
	var sql = `
		insert into ${table} ( ${ Object.keys(o).join(',') } )
		values ( ${ Object.keys(o).map( (_, i) => '$'+(i+1+args.length)) } )
	`;
	if( condition ){
		sql += ` where ${condition} `;
	}
	sql += `returning *`;
	return plv8.execute(sql, args.concat(Object.keys(o).map( k => o[k] )) );
}

// Helper for simple updates:
// db.update('mytable', {col1:'hello', col2:true});
// db.update('mytable', {col1:'hello', col2:true}, `col1 IS NULL`);
// db.update('mytable', {col1:'hello', col2:true}, `id = $1 AND name = $2`, id, name);
// ... for anything more complex, use db.query()
export function update(table, o, condition, ...args){
	var keyvals = Object.keys(o).map( (k, i) => {
		return [k, '$'+(i+1+args.length)].join(' = ')
	}).join(', ');
	var values = Object.keys(o).map( k => o[k] )
	var sql = `
		update ${table}
		set ${keyvals}
	`;
	if( condition ){
		sql += ` where ${condition} `;
	}
	sql += `returning *`;
	return plv8.execute(sql, args.concat(values) );
}

// Helper for simple deletes:
// db.destroy('mytable');
// db.destroy('mytable', `col1 IS NULL`);
// db.destroy('mytable', `id = $1`, id);
// ... for anything more complex, use db.query()
export function destroy(table, condition, ...args){
	var sql = `
		delete from ${table}
	`;
	if( condition ){
		sql += ` where ${condition} `;
	}
	sql += `returning *`;
	return plv8.execute(sql, args);
}

export function count(sql, ...args){
	return query(`WITH x AS (${sql}) SELECT count(*) AS c FROM x`, ...args)[0].c;
}

export default {
	query,
	transaction,
	insert,
	update,
	destroy,
	count
}
