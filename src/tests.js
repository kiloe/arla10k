var alice, bob, jeff, kate;

function createMember( o ) {
	var id = q( 'select create_member(gen_random_uuid(), $1) as id', o.email )[ 0 ].id;
	delete o.email;
	for ( let k in o ) {
		q( `update member set ${ k } = $1 where id = $2`, o[ k ], id );
	}
	var member = q( 'select * from member where id = $1', id )[ 0 ];
	member.member_email = q( 'select * from member_email where member_id = $1', id )[ 0 ];
	return member;
}

check( "we can create members", function () {
	// alice is member with a generic email address
	alice = createMember( {
		first_name: ' Alice  ',
		last_name: ' Memberton		',
		email: 'alice@GMAIL.com',
	} );
	// kate is member with a generic email address
	kate = createMember( {
		first_name: 'Kate',
		last_name: 'Userton',
		email: 'kate@gmail.com',
	} );
	// bob is a member with what looks like a professional email address
	bob = createMember( {
		first_name: 'Bob',
		last_name: 'Userman',
		is_su: true,
		email: 'bob@exampleclinic.com',
	} );
	// jeff is a member with an email address like bob's
	jeff = createMember( {
		first_name: 'Jeff',
		last_name: 'Userman',
		email: 'jeff@exampleclinic.com',
	} );
} )

check( "there is at least one superuser member", function () {
	assert( count( 'select * from member where is_su' ) > 0 );
} )

check( "member names are trimmed", function () {
	assert.equal( alice.first_name, 'Alice' );
	assert.equal( alice.last_name, 'Memberton' );
} )

check( "member email addresses are lowercased and trimmed", function () {
	assert.equal( alice.member_email.address, 'alice@gmail.com' );
} )

checkFail( "deleting the last email address of a member is impossible", function () {
	q( 'delete from member_email where member_id = $1', bob.id );
} )

checkFail( "deleting the last member is impossible", function () {
	q( 'delete from member' );
} )

check( "4 orgs exist so far (the 'personal' orgs)", function () {
	assert.equal( count( 'select * from org' ), 4 );
} )

check( "email can be confirmed", function () {
	q( 'update member_email set is_confirmed = true' );
} );

check( "there is no 'gmail.com' org as it should be ignored", function () {
	assert.equal( count( "select * from org where domain = 'gmail.com'" ), 0 );
} )

check( "company orgs were automatically created as a result of confirming emails", function () {
	assert.equal( count( 'select * from org' ), 5 );
} )

check( "alice is the sole admin of her of her own private org", function () {
	var membership = q( 'select * from org_member where member_id = $1', alice.id );
	assert( membership.length == 1, 'expected only one membership' );
	assert( membership[ 0 ].is_admin === true, 'expected alice to be an admin' );
	alice.orgs = q( 'select org.*, org_member.is_admin from org, org_member where org.id = org_member.org_id and org_member.member_id = $1', alice.id );
} )

check( "bob is the sole admin of a company org with two members", function () {
	bob.orgs = q( 'select org.*, org_member.is_admin from org, org_member where org.id = org_member.org_id and org_member.member_id = $1 and org.is_company = true', bob.id );
	assert.equal( bob.orgs.length, 1 );
	assert( bob.orgs[ 0 ].is_admin, 'is admin' );
	assert( count( 'select * from org_member where org_id = $1', bob.orgs[ 0 ].id ) == 2, 'only two member' );
	assert( count( 'select * from org_member where org_id = $1 and is_admin', bob.orgs[ 0 ].id ), 1, 'only one admin' );
} )

check( "jeff is a non-admin member of the same org as bob", function () {
	jeff.orgs = q( 'select org.*, org_member.is_admin from org, org_member where org.id = org_member.org_id and org_member.member_id = $1 and org.is_company = true', jeff.id );
	assert( jeff.orgs.length == 1, 'expected one company org' );
	assert( jeff.orgs[ 0 ].is_admin === false, 'not admin' );
} )

check( "members can be invited to orgs", function () {
	insert( 'org_member', {
		org_id: q( 'select org.id from org, org_member where org.id = org_member.org_id and org_member.member_id = $1', bob.id )[ 0 ].id,
		member_id: kate.id,
	} )
} )

check( "kate is a member of her own org AND bob's too", function () {
	var kateOrgs = q( 'select org.* from org, org_member where org_member.org_id = org.id and org_member.member_id = $1', kate.id );
	var bobsOrgs = q( 'select org.* from org, org_member where org_member.org_id = org.id and org_member.member_id = $1', bob.id );
	assert.equal( kateOrgs.length, 2 );
	assert( bobsOrgs.some( a => kateOrgs.some( b => a.id == b.id ) ), 'kate is member of bobs org' );
} )

check( "cases belong to clients", function () {
	q( 'select "case".* from "case", client where "case".client_id = client.id' );
} );

// graphql

check( "graphql works", function () {
	assert.deepEqual( q( 'select graphql($1);', `
		member(${ alice.id }) {
			id,
			first_name,
			email_addresses.first(1) {
				id,
				address,
			},
			orgs {
				id,
			}
		}
	` )[ 0 ].graphql, {
		member: {
			id: alice.id,
			first_name: alice.first_name,
			email_addresses: [ {
				id: alice.member_email.id,
				address: alice.member_email.address
			} ],
			orgs: [ {
				id: alice.orgs[ 0 ].id,
			} ]
		}
	} );
} )
