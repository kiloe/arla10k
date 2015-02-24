/*

NOTE: THIS IS JUST AN EXAMPLE APP FOR TESTING!
During normal usage this "app" directory is overwritten (or bind-mounted)
from the user's app.

*/

export function create_member( id, email ) {
	return this.query(`
		with m as (
			insert into member (id) values ($1) returning *
		) insert into member_email
			(member_id, address, is_confirmed)
			(select m.id, $2, false from m)
			returning member_id as id;
	`, id, email)[0].id;
};

export function find_member( reference ) {
	var r = this.query(`
		select member_id from member_email where address = lower($1) limit 1
	`, reference)[0];
	if( !r ){
		return null;
	}
	return r.member_id;
};
