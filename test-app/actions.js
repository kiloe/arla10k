
export function registerMember( {username, password} ) {
	return [`
		insert into member (username, password)
		values ($1, $2)
	`, username, password];
};

export function exampleOp( a, b, c ) {
	return [`select $1,$2,$3`, a, b, c];
};
