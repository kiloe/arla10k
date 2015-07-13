
export class root {

	static me = {type: 'member', query: function(){
		return [`select * from member where id = $1`, this.session.id];
	}}

	static members = {type: 'array', of: 'member', query: function(){
		return `select * from member`;
	}}

	static email_addresses = {type: 'array', of: 'email', query: function(){
		return `select * from email`;
	}}

	static numbers = {type:'array', of:'int', query: function(){
		return `select * from unnest(ARRAY[10,5,11])`
	}}

}

export class member {

	// simple properties
	static username             = {type: 'text'}
	static password             = {type: 'text'}
	static is_su                = {type: 'boolean', def: 'false' }

	// common one-to-many style relationship
	static email_addresses = {type: 'array', of: 'email', query: function(){
		return [`select * from email where member_id = $1`, this.id];
	}}

	// self referencing many-to-many ... this could get real expensive real fast
	static friends = {type: 'array', of: 'member', query: function(){
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
	}}

	// contrived example of a computed property
	static uppername = {type: 'text', query: function(){
		return [`select upper(username) from member where id = $1`, this.id]
	}}

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
	static member_id           = {type: 'uuid', ref: 'member'}
	static addr                = {type: 'text', unique:true}

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
	static member_1_id = {type: 'uuid', ref:'member'}
	static member_2_id = {type: 'uuid', ref:'member'}

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
