/*

NOTE: THIS IS JUST AN EXAMPLE APP FOR TESTING!
During normal usage this "app" directory is overwritten (or bind-mounted)
from the user's app.

*/

export function createIdentity( values ) {
	return [`
		with m as (
			insert into member (id) values ($1) returning *
		) insert into member_email
			(member_id, address, is_confirmed)
			(select m.id, $2, false from m)
			returning member_id as id;
	`, values.id, values.email];
};

export function lookupIdentity( reference ) {
	return [`
		select member_id from member_email where address = lower($1) limit 1
	`, reference];
};


export function exampleOp( a, b, c ) {
	return [`select $1,$2,$3`, a, b, c];
};
