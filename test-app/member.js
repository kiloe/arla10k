import email from './email';

export default class member extends arla.Entity {

	static requires = {
		email,
	}

	static props = {
		// simple properties
		name:                {type: 'text', nullable:true},
		username:             {type: 'text'},
		password:             {type: 'text'},
		is_su:                {type: 'boolean', def: 'false' },

		// common one-to-many style relationship
		email_addresses: {type: 'array', of: email, query: function(){
			return [`select * from ${email} where member_id = $1`, this.id];
		}},

		// self referencing many-to-many ... this could get real expensive real fast
		friends: {type: 'array', of: member, query: function(){
			return [`
				select m2.id, m2.username from member m1
				left join friend
					on m1.id = friend.member_1_id
					or m1.id = friend.member_2_id
				left join member m2
					on m2.id = friend.member_1_id
					or m2.id = friend.member_2_id
				where m1.id = $1 and m2.id != $1
			`,this.id];
		}},

		// contrived example of a computed property
		uppername: {type: 'text', query: function(){
			return [`select upper(username) from member where id = $1`, this.id];
		}},

		// return everyone! (but it may be shadowed by parent)
		everyone: {type: 'array', of:member, query:function(){
			return [`
				select * from member
			`];
		}}
	}

	// after update or insert trigger
	afterChange() {
		if( this.username.length < 3 ){
			throw new UserError('username too short');
		}
	}

	// before delete trigger
	beforeDelete(){
		if( this.username == 'alice' ){
			throw 'alice is indestructable!';
		}
	}

}
