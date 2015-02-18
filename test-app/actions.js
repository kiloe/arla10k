/*

NOTE: THIS IS JUST AN EXAMPLE APP FOR TESTING!
During normal usage this "app" directory is overwritten (or bind-mounted)
from the user's app.

*/

import db from "../db";

export function create_member( id, email ) {
	return db.query(`
		with m as (
			insert into member (id) values ($1) returning *
		) insert into member_email
			(member_id, address, is_confirmed)
			(select m.id, $2, false from m)
			returning member_id as id;
	`, id, email)[0].id;
};

export function create_thing( db, [id, email] ) {
	return db.query(`
		with m as (
			insert into member (id) values ($1) returning id
		)
		insert into member_email
			(member_id, address, is_confirmed)
			(select m.id, $2, false from m)
			returning member_id as id
	`, id, email);
};
