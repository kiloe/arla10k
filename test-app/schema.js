import member from './member';
import email from './email';


export class friend extends arla.Entity {
	static props = {
		member_1_id: {ref:member},
		member_2_id: {ref:member},
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

import country from './country';
export class root extends arla.Entity {

	static requires = {
		country
	}

	static with(){
		return [`shadowed_members as (select id,username from member where length(username) < 4)`];
	}

	static props = {
		me: {type: member, query: function(){
			return [`select * from ${member} where id = $1`, this.session.id];
		}},

		members: {type: 'array', of: member, query: function(){
			return `select * from ${member}`;
		}},

		email_addresses: {type: 'array', of: email, query: function(){
			return `select * from ${email}`;
		}},

		numbers: {type:'array', of:'int', query: function(){
			return `select * from unnest(ARRAY[10,5,11])`;
		}},

		// country data is populated in arla.configure via bootstrap
		countries: {type: Array, of: 'country', query: function(){
			return `select * from ${country}`;
		}},

		// someflag should be set from the authentication function in arla.configure
		someflag: {type: Boolean, query: function(){
			return [`select $1`, this.session.someflag];
		}}
	}

}

