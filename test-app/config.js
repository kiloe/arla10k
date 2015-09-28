import * as actions from './actions';
import * as schema from './schema';

arla.configure({
	// set the API version
	version: 2,
	// actions declares the mutation functions that are exposed
	actions: actions,
	// schema is an Object that declares the struture of your data
	// and how queries should be built.
	schema: schema,
	// the authenticate function accepts user credentials and returns
	// the query that will return the values that will be used as the
	// context/claims/session for future requests
	authenticate({username, password}){
		return [`
			select id from member
			where username = $1
			and password = crypt($2, password)
		`, username, password];
	},
	// the register function returns the mutation-action action that will
	// be executed to register a new user.
	// The reason for this transformation is to prevent the password from
	// ending up in the mutation log
	register(values){
		values.password = pgcrypto.crypt(values.password);
		return {
			name: 'registerMember',
			args:[values]
		};
	},
	// transform is called when a mutation with a version < the one set above
	// is executed. It allows you to change your API while staying backwards
	// compatible and gracefully handling data migration in a lossless way.
	//
	// In this example we transform an old "createUser" mutation with version 1
	// into a "registerMember" mutation with version 2.
	//
	// transform is called as many times as required until a mutation with the
	// current version number is returned
	transform(m, targetVersion){
		switch(m.name){
			case 'createUser':
				switch(m.version){
					case 1: return {
						name: 'registerMember',
						args: [{
							id: m.args[0],
							username: m.args[1],
							password: pgcrypto.crypt(m.args[2])
						}],
						version: 2
					};
				}
		}
		return Object.assign(m, {version: targetVersion});
	},
	// bootstrap is an optional array of SQL statements to execute before any
	// mutations are replayed.
	// This allows you to setup the database, install extensions and setup any
	// required data (like the initial users).
	// Note: it's a fair bit slower than replaying mutations, so don't go nuts or
	// you risk increasing app startup time.
	bootstrap: [
		`insert into country (name,code) values ('United Kingdom', 'GB')`,
		[`insert into country (name,code) values ($1, $2)`, 'France', 'FR'],
	],

});
