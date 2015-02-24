import db from "../db";
import assert from "../assert";
import {define, defineJoin} from "../runtime";
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

define('member', {
	properties: {
		first_name: {type: 'text',      def: "''"},
		last_name:  {type: 'text',      def: "''" },
		is_su:      {type: 'boolean',   def: 'false' }
	},
	refs: {
		email_addresses: {
			hasMany: 'member_email'
		},
		orgs: {
			hasMany: 'org',
			via: 'org_member'
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
})

define('member_email', {
	properties: {
		address: { type: 'text', unique: true },
		is_confirmed: { type: 'boolean' },
	},
	refs: {
		member: {
			hasOne: 'member'
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
});

define('org', {
	properties: {
		name: { type: 'text' },
		domain: { type: 'text', nullable: true, def: null },
		is_company: { type: 'boolean' },
	},
	refs: {
		members: {
			hasMany: 'member',
			via: 'org_member'
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
});

defineJoin([ 'org', 'member' ], {
	properties: {
		is_admin: { type: 'boolean' },
		is_confirmed: { type: 'boolean' },
	}
} );

define( 'team', {
	properties: {
		name: { type: 'text' },
	},
} );

defineJoin( [ 'team', 'member' ], {
	properties: {
		is_admin: { type: 'text' },
	}
} );

define( 'client', {
	properties: {
		first_name: { type: 'text' },
		last_name: { type: 'text' },
	},
} );

define( 'issue', {
	properties: {
		name: { type: 'text' },
	},
} );

define( 'exercise', {
	properties: {
		name: { type: 'text' },
	},
} );

define( 'case', {
	properties: {
		client_id: { type: 'uuid', ref: 'client' },
		org_id: { type: 'uuid', ref: 'org' },
		issue_id: { type: 'uuid', ref: 'issue' },
	}
} );

defineJoin( [ 'case', 'member' ], {
	properties: {
		is_admin: { type: 'boolean' },
	}
} );

defineJoin( [ 'case', 'team' ] );

define( 'course', {
	properties: {
		case_id: { type: 'uuid', ref: 'case' },
		name: { type: 'text' },

	},
} );

define( 'assignment', {
	properties: {
		course_id: { type: 'uuid', ref: 'course' },
		name: { type: 'text' },
		due_at: { type: 'timestamptz' },
	},
} );

define( 'task', {
	properties: {
		assignment_id: { type: 'uuid', ref: 'assignment' },
		execise_id: { type: 'uuid', ref: 'exercise', nullable: true },
		name: { type: 'text' },
		is_complete: { type: 'boolean', nullable: true, def: null },
		instruments: { type: 'json' },
	},
} );

define('data', {
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
} );
