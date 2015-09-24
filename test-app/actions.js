
export function registerMember( {id, name, username, password} ) {
	return [`
		insert into member (id, name, username, password)
		values ($1, $2, $3, $4)
	`, id, name, username, password];
}

export function destroyMember() {
	return [`
		delete from member where id = $1
	`, this.session.id];
}

export function addEmailAddress(addr) {
	return [`
		insert into email (member_id, addr)
		values ($1, $2)
	`, this.session.id, addr];
}

export function updateEmailAddress(oldAddr, newAddr) {
	return [`
		update email set addr = $1
		where addr = $2 and member_id = $3
	`, newAddr, oldAddr, this.session.id];
}

export function addFriend(id) {
	return [`
		insert into friend (member_1_id, member_2_id)
		values ($1, $2)
	`, this.session.id, id];
}

export function exampleOp( a, b, c ) {
	return [`select $1,$2,$3`, a, b, c];
}
