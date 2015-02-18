var request = require('supertest'),
	expect = require('chai').expect,
	Promise = require('es6-promise').Promise,
	secret = 'testing',
	app = require('./helper/app'),
	uuid = require('node-uuid');

describe('graphql', function(){

	var user, token;

	before(function(done){
		var id = uuid.v4();
		var u ={ 
			id: id,
			first_name: 'user',
			last_name: id.toString(),
			email: id+'@test.com',
			password: 'th1s-1s-4-p4ssword-'+id 
		};
		request(app)
			.post('/register')
			.send(u)
			.end(function(err, res){
				if( err ){
					return done(err);
				}
				user = u;
				token = res.body.access_token;
				done();
			})
	})

	it('should fetch member', function(done){
		request(app)
			.post('/query')
			.set('Authorization', 'Bearer '+token)
			.set('Content-Type', 'text/plain')
			.send("member("+user.id+'){ id }')
			.end(function(err, res){
				if( err ){
					return done(err)
				}
				expect(res.body).to.deep.equal({ member: { id: user.id } })
				done();
			})
	})
})
