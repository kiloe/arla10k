
export function registerMember( {id, username, password} ) {
	return [`
		insert into member (id, username, password)
		values ($1, $2, $3)
	`, id, username, password];
};

export function addEmailAddress(addr) {
	return [`
		insert into email (member_id, addr)
		values ($1, $2)
	`, this.session.id, addr];
}

export function addFriend(id) {
	return [`
		insert into friend (member_1_id, member_2_id)
		values ($1, $2)
	`, this.session.id, id];
}

export function exampleOp( a, b, c ) {
	return [`select $1,$2,$3`, a, b, c];
};
