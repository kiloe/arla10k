
export class root {

	static me = {type: member, query: function(){
			return [`select * from member where id = $1`, this.current_user.id]
	}}

	static members = {type: 'array', of: 'member', query: function(){
		return `select * from member`
	}}

}

export class email {
		static address = {type: ''}
}

export class member {

	static username             = {type: 'text'}
	static password             = {type: 'text'}
	static is_su                = {type: 'boolean',   def: 'false' }
	static email_addresses      = {type: 'array', of: 'email', query: function(){
		return `select * from email`;
	}}

	afterChange() {
		if( this.username.length < 3 ){
			throw new Error('username too short');
		}
	}


}
