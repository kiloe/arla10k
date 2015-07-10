
export class root {

	static me = {type: 'member', query: function(){
		return [`select * from member where id = $1`, this.session.id]
	}}

	static members = {type: 'array', of: 'member', query: function(){
		return `select * from member`
	}}

}

export class member {

	static username             = {type: 'text'}
	static password             = {type: 'text'}
	static is_su                = {type: 'boolean',   def: 'false' }
	static email_addresses      = {type: 'array', of: 'email', query: function(){
		return [`select * from email where member_id = $1`, this.id];
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
}
