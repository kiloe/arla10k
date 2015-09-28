
export class member {

	static props = {
		// simple properties
		name:                {type: 'text', nullable:true},
		username:             {type: 'text'},
		password:             {type: 'text'},
		is_su:                {type: 'boolean', def: 'false' },

		// common one-to-many style relationship
		email_addresses: {type: 'array', of: 'email', query: function(){
			return [`select * from email where member_id = $1`, this.id];
		}},

		// self referencing many-to-many ... this could get real expensive real fast
		friends: {type: 'array', of: 'member', query: function(){
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

export class email {

	static props = {
		member_id:           {type: 'uuid', ref: member},
		addr:                {type: 'text', unique:true}
	}

	beforeChange(){
		this.validateEmail();
	}

	validateEmail(){
		if( !(/.@./.test(this.addr)) ){
			throw new UserError(this.addr+' does not appear to be a valid email address');
		}
		this.addr = this.addr.toLowerCase().trim();
	}
}

export class friend {
	static props = {
		member_1_id: {type: 'uuid', ref:member},
		member_2_id: {type: 'uuid', ref:member},
	}

	static indexes = {
		unique_join: {on: ['member_1_id', 'member_2_id'], unique:true}
	}

	beforeChange(){
		if( this.member_1_id > this.member_2_id ){
			[this.member_1_id,this.member_2_id] = [this.member_2_id, this.member_1_id];
		}
		if( this.member_1_id == this.member_2_id ){
			throw new UserError('you cannot make friends with yourself');
		}
	}
}

export class root {

	static props = {
		me: {type: member, query: function(){
			return [`select * from member where id = $1`, this.session.id];
		}},

		members: {type: 'array', of: member, query: function(){
			return `select * from member`;
		}},

		email_addresses: {type: 'array', of: email, query: function(){
			return `select * from email`;
		}},

		numbers: {type:'array', of:'int', query: function(){
			return `select * from unnest(ARRAY[10,5,11])`;
		}},

		shadowed_members: {type: 'array', of:member, query: function(){
			return {
				with: `member as (select id,username from member where length(username) < 4)`,
				query: `select * from public.member`,
				args: []
			};
		}}
	}

}

