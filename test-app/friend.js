
export default class friend extends arla.Entity {
	static props = {
		member_1_id: {ref:'member'},
		member_2_id: {ref:'member'},
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
