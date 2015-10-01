
export default class country extends arla.Entity {
	static props = {
		id:   {type:'uuid', pk:true},
		name: {type: String},
		code: {type: String},
	}
}
