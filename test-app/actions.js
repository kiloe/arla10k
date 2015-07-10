
export function registerMember( {username, password} ) {
	return [`
		insert into member (username, password)
		values ($1, $2)
	`, username, password];
};

export function addEmailAddress(addr) {
	return [`
		insert into email (member_id, addr)
		values ($1, $2)
	`, this.session.id, addr];
}

export function exampleOp( a, b, c ) {
	return [`select $1,$2,$3`, a, b, c];
};
