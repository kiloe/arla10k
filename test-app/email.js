
export default class email extends arla.Entity {

	static props = {
		member_id:           {ref: 'member'},
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
