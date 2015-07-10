
export class root {

	static me = {type: 'member', query: function(){
		return [`select * from member where id = $1`, this.session.id];
	}}

	static members = {type: 'array', of: 'member', query: function(){
		return `select * from member`;
	}}

}

export class member {

	static username             = {type: 'text'}
	static password             = {type: 'text'}
	static is_su                = {type: 'boolean', def: 'false' }

	static email_addresses = {type: 'array', of: 'email', query: function(){
		return [`select * from email where member_id = $1`, this.id];
	}}

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
		`, this.id];
	}}

	afterChange() {
		if( this.username.length < 3 ){
			throw new Error('username too short');
		}
	}
}

export class email {
	static member_id           = {type: 'uuid', ref: 'member'}
	static addr                = {type: 'text'}

	beforeChange(){
		this.validateEmail();
	}

	validateEmail(){
		if( !(/.@./.test(this.addr)) ){
			throw new Error(this.addr+' does not appear to be a valid email address');
		}
		this.addr = this.addr.toLowerCase().trim();
	}
}

export class friend {
	static member_1_id = {type: 'uuid', ref:'member'}
	static member_2_id = {type: 'uuid', ref:'member'}

	beforeChange(){
		if( this.member_1_id > this.member_2_id ){
			this.member_1_id,this.member_2_id = this.member_2_id, this.member_1_id;
		}
	}
}
