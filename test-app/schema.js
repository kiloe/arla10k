import db from "../db";
import assert from "../assert";
/*

NOTE: THIS IS JUST AN EXAMPLE APP FOR TESTING!
During normal usage this "app" directory is overwritten (or bind-mounted)
from the user's app.

*/


function constraint(name, fn){
	try{
		fn();
	} catch(err) {
		throw name + ': ' + err;
	}
}

// The "root" entity is a special psuedo entity decalres the root
// calls.
// All queries start here.
export var root = {
	edges: {
		viewer() {
			return {
				type: 'member',
				query: `select * from member where id = $viewer limit 1`
			}
		}
	}
}

export var member = {
	properties: {
		first_name: {type: 'text',      def: "''"},
		last_name:  {type: 'text',      def: "''" },
		is_su:      {type: 'boolean',   def: 'false' },
	},
	edges: {
		email_addresses() {
			return {
				type: 'array',
				of: 'email_address',
				query: `
					select * from email_address
					where email_address.member_id = $id`
				,
			}
		},
		orgs() {
			return {
				type: 'array',
				of: 'org',
				query: `
					select org.* from org_member
					left join org on org_member.org_id = org.id
					where org_member.member_id = $this.id
				`
			}
		},
	},
	beforeChange( r ) {
		if ( r.first_name ) {
			assert( r.first_name.length >= 2, "first name must be at least 2 characters long" );
		}
		if ( r.last_name ) {
			assert( r.last_name.length >= 3, "last name must be at least 3 characters long" );
		}
	},
	afterInsert( r ) {
		constraint( "new users should always have an email address record", function () {
			assert( db.count( 'select * from member_email WHERE member_id = $1', r.id ) > 0 );
		} );
		var org = db.insert( 'org', {
			id: r.id,
			name: "Personal"
		} )[0];
		var membership = db.insert( 'org_member', {
			id: r.id,
			org_id: org.id,
			member_id: r.id,
			is_admin: true,
		} )[0];
	}
}

export var member_email = {
	properties: {
		member_id: {type: 'uuid', ref:'member'},
		address: { type: 'text', unique: true },
		is_confirmed: { type: 'boolean' },
	},
	edges: {
		member() {
			return {
				type: 'member',
				query: [`
					select * from member where id = $1
				`, this.member_id]
			}
		},
	},
	beforeInsert( r ) {
		constraint( "member_email validation", function () {
			r.address = ( r.address || '' ).trim().toLowerCase();
			assert( /.+@.+\..+/.test( r.address ), "email address does not look valid" );
		} )
		constraint( "blacklisted domain", function () {
			var blacklist = [
				/mailinator/
			];
			assert( !blacklist.some( m => m.test( r.address ) ), "cannot accept this email address" )
		} )
	},
	beforeUpdate( r, e ) {
		constraint( "member_email records are immutable except for confirmation field", function () {
			assert( r.address == e.old.address, 'cannot alter email address field' );
		} )
	},
	afterChange( r ) {
		// attempt to join or create any orgs that match this email address
		if ( r.is_confirmed ) {
			var domain = r.address.split( '@' )[ 1 ];
			var sharedDomain = [
				/gmail|yahoo|hotmail|outlook|msn|qq.com/
			]
			if ( !sharedDomain.some( m => m.test( domain ) ) ) {
				var org = db.query( 'select * from org where domain = $1', domain )[ 0 ];
				if ( !org ) {
					org = db.insert( 'org', {
						id: r.id,
						domain: domain,
						is_company: true,
					} )[0];
				}
				var membership = db.query( 'select * from org_member where org_id = $1 and member_id = $2', [ org.id, r.member_id ] )[ 0 ];
				if ( !membership ) {
					membership = db.insert( 'org_member', {
						id: r.id,
						org_id: org.id,
						member_id: r.member_id,
						is_admin: db.count( 'select * from org_member where org_id = $1', org.id ) === 0,
					} )[0]
				}
			}
		}
	},
	afterDelete( r ) {
		assert(
			db.count( 'select * from member WHERE id = $1', r.member_id ) == 0 ||
			db.count( 'select * from member_email WHERE member_id = $1', r.member_id ) > 0,
		'action would have left member without an email address' );
	},
};

export var org = {
	properties: {
		name: { type: 'text' },
		domain: { type: 'text', index:true, nullable: true, def: null },
		is_company: { type: 'boolean' },
	},
	edges: {
		members() {
			return {
				type: 'array',
				of: 'member',
				query: `
					select member.*
					from org_member
					left join member on member.id = org_member.member_id
					where org_member.org_id = $this.id
				`
			}
		},
	},
	indexes: {
		domain: {
			unique: true,
			on: [ 'domain' ]
		}
	},
	beforeChange( r ) {
		constraint( "org validation", function () {
			if ( r.domain ) {
				assert( /\./.test( r.domain ), "domain name does not appear to be valid" );
				if ( !r.name ) {
					r.name = `${ r.domain }'s org`;
				}
			}
		} )
	},
};

export var org_member = {
	properties: {
		org_id: {type: 'uuid', ref:'org'},
		member_id: {type: 'uuid', ref:'member'},
		is_admin: { type: 'boolean' },
		is_confirmed: { type: 'boolean' },
	}
};

export var team = {
	properties: {
		name: { type: 'text' },
	},
};

export var team_member = {
	properties: {
		team_id: {type: 'uuid', ref:'team'},
		member_id: {type: 'uuid', ref:'member'},
		is_admin: { type: 'text' },
	}
};

export var client = {
	properties: {
		first_name: { type: 'text' },
		last_name: { type: 'text' },
	},
};

export var issue = {
	properties: {
		name: { type: 'text' },
	},
};

export var exercise = {
	properties: {
		name: { type: 'text' },
	},
};

export var client_case = {
	properties: {
		client_id: { type: 'uuid', ref: 'client' },
		org_id: { type: 'uuid', ref: 'org' },
		issue_id: { type: 'uuid', ref: 'issue' },
	}
};

export var case_member = {
	properties: {
		is_admin: { type: 'boolean' },
	}
};

export var case_team = {
	properties: {
		case_id: {type: 'uuid', ref: 'client_case'},
		team_id: {type: 'uuid', ref: 'team'}
	}
};

export var course = {
	properties: {
		case_id: { type: 'uuid', ref: 'client_case' },
		name: { type: 'text' },
	},
};

export var assignment = {
	properties: {
		course_id: { type: 'uuid', ref: 'course' },
		name: { type: 'text' },
		due_at: { type: 'timestamptz' },
	},
};

export var task = {
	properties: {
		assignment_id: { type: 'uuid', ref: 'assignment' },
		execise_id: { type: 'uuid', ref: 'exercise', nullable: true },
		name: { type: 'text' },
		is_complete: { type: 'boolean', nullable: true, def: null },
		instruments: { type: 'json' },
	},
};

export var data = {
	properties: {
		created_at: {type: 'timestamptz', def:'now()'},
		task_id: { type: 'uuid', ref: 'task' },
		value: { type: 'json' },
	},
	indexes: {
		created_at: {
			on: [ 'created_at' ]
		}
	}
};
