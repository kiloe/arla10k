var request = require('supertest'),
	expect = require('chai').expect,
	jwt = require('jwt-simple'),
	moment = require('moment'),
	Promise = require('es6-promise').Promise,
	secret = 'testing',
	app = require('./helper/app'),
	uuid = require('node-uuid'),
	errors = require('../errors');

function generateRandomUser(){
	return new Promise(function(resolve, reject){
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
			.expect('Content-Type', /json/)
			.end(function(err, res){
				if( err ){
					return reject(err);
				}
				expect(res.body).to.have.property('access_token');
				var info = jwt.decode(res.body.access_token, secret);
				expect(info).to.be.a('object');
				expect(info).to.have.property('exp');
				expect(info).to.have.property('id');
				resolve(u);
			})
	});
}

function registerRandomUsers(n, done){
	var users = [];
	var registrations = [];
	for(var i=0; i<n; i++){
		registrations[i] = function(){
			return generateRandomUser();
		}
	}
	registrations.reduce(function(p, fn){
		return p.then(fn);
	}, Promise.resolve()).then(function(lastUser){
		done(null, lastUser);
	}).catch(done);
}

describe('ident', function(){

	var u;  // a random valid user

	before(function(done){
		registerRandomUsers(1, function(err, user){
			if( err ){
				return done(err)
			}
			u = user;
			expect(u).to.have.property('id');
			expect(u).to.have.property('email');
			expect(u).to.have.property('password');
			done();
		});
	});

	describe('registration', function(){

		it('should reject blank passwords', function(done){
			request(app)
				.post('/register')
				.send({
					id: uuid.v4(),
					first_name: 'bob',
					last_name: 'bob',
					email: 'bob@bob.com',
					password: ''
				})
				.expect('Content-Type', /json/)
				.expect(function(res){
					expect(res.body).to.have.property('errors')
					expect(res.body.errors[0]).to.equal(errors.PasswordTooShort)
				})
				.end(done)
		})

		it('should reject passwords under 9 characters', function(done){
			request(app)
				.post('/register')
				.send({
					id: uuid.v4(),
					first_name: 'bob',
					last_name: 'bob',
					email: 'bob@bob.com',
					password: 'bd73jsyd'
				})
				.expect('Content-Type', /json/)
				.expect(400)
				.expect(function(res){
					expect(res.body).to.have.property('errors')
					expect(res.body.errors).to.be.an.Array;
					expect(res.body.errors[0]).to.equal(errors.PasswordTooShort)
				})
				.end(done)
		})

		it('should reject short passwords without non-aplhanumeric character',function(done){
			request(app)
				.post('/register')
				.send({
					id: uuid.v4(),
					first_name: 'bob',
					last_name: 'bob',
					email: 'bob@bob.com',
					password: 'bdjsydggygyg'
				})
				.expect('Content-Type', /json/)
				.expect(400)
				.expect(function(res){
					expect(res.body).to.have.property('errors')
					expect(res.body.errors).to.be.an.Array;
					expect(res.body.errors[0]).to.equal(errors.PasswordTooSimple)
				})
				.end(done)
		})

		it('should reject purely numeric passwords',function(done){
			request(app)
				.post('/register')
				.send({
					id: uuid.v4(),
					first_name: 'bob',
					last_name: 'bob',
					email: 'bob@bob.com',
					password: '023975029375'
				})
				.expect('Content-Type', /json/)
				.expect(400)
				.expect(function(res){
					expect(res.body).to.have.property('errors')
					expect(res.body.errors).to.be.an.Array;
					expect(res.body.errors[0]).to.equal(errors.PasswordTooNumeric)
				})
				.end(done)
		})


		it('should comfortably create 5 users within 2s', function(done){
			registerRandomUsers(5, done);
		})

		it('should be rate limited by ip')

	})

	describe('login', function(){

		it('should respond with json', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.email, password: u.password })
				.expect('Content-Type', /json/)
				.end(done)
		})

		it('should return an access_token with id/password auth', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.id, password: u.password })
				.expect(function(res){
						expect(res.body).to.have.property('access_token')
				})
				.end(done);
		})

		it('should return a refresh_token', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.id, password: u.password })
				.expect(function(res){
						expect(res.body).to.have.property('refresh_token')
				})
				.end(done);
		})

		it('should allow email to be used in place of id', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.email, password: u.password })
				.expect(function(res){
						expect(res.body).to.have.property('access_token')
				})
				.end(done);
		})

		it('should not be case-sensitive for email', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.email.toUpperCase(), password: u.password })
				.expect(function(res){
						expect(res.body).to.have.property('access_token')
				})
				.end(done);
		})

		it('should fail when incorrect password used', function(done){
			request(app)
				.post('/auth')
				.send({ username: u.id, password: 'bad' })
				.expect('Content-Type', /json/)
				.expect(403)
				.expect(function(res){
						expect(res.body).to.have.property('errors');
						expect(res.body.errors[0]).to.equal(errors.InvalidPassword);
						expect(res.body).to.not.have.property('access_token');
				})
				.end(done)
		})

		it('should fail when incorrect id used', function(done){
			request(app)
				.post('/auth')
				.send({ username: 'bad', password: u.password })
				.expect('Content-Type', /json/)
				.expect(403)
				.expect(function(res){
						expect(res.body).to.have.property('errors');
						expect(res.body.errors[0]).to.equal(errors.InvalidUserId);
				})
				.end(done)
		})

	})


	describe("access_token", function(){

		var token;

		before(function(done){
			request(app)
				.post('/auth')
				.send({ username: u.email, password: u.password })
				.expect('Content-Type', /json/)
				.expect(function(res){
					expect(res.body).to.have.property('access_token');
					token = jwt.decode(res.body.access_token, secret);
				})
				.end(done)
		})

		it('should be an object', function(){
			expect(token).to.be.an.object;
		})

		it('should have type=access', function(){
			expect(token).to.have.property('type');
			expect(token.type).to.equal('access');
		})

		it('should contain a user uuid', function(){
			expect(token).to.have.property('id');
			expect(token.id).to.be.a.string;
			expect(token.id).to.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
		})

		it('should expire after 2 hours', function(){
			var now = moment();
			expect(token).to.have.property('exp');
			expect(moment(token.exp).isAfter(now)).to.be.true;
			expect(moment(token.exp).isBefore(now.add(3, 'hours'))).to.be.true;
		})

	})



	// Ghosting is where an admin user can assume the identity of another
	// (non-admin) user, usually for the purpose of debugging an issue or
	// making changes on behalf of the 'ghosted' user
	describe('ghosting', function(){

		var su, u2;

		// create a superuser
		before(function(){
			return generateRandomUser().then(function(user){
				su = user;
				app.su = su.id;
				return generateRandomUser().then(function(user){
					u2 = user;
				})
			})
		});

		it('should return a ghosted token if user is valid and is a superuser', function(done){
			request(app)
				.post('/auth')
				.send({ username: su.id, password: su.password, as: u.id })
				.expect(200)
				.expect(function(res){
						expect(res.body).to.have.property('access_token');
						var token = jwt.decode(res.body.access_token, secret);
						expect(token.type).to.equal('access');
						expect(token.id).to.equal(u.id);
				})
				.end(done);
		})

		it('should ignore the "as" parameter if user is not a superuser', function(done){
			request(app)
				.post('/auth')
				.send({ username: u2.id, password: u2.password, as: u.id })
				.expect(200)
				.expect(function(res){
						expect(res.body).to.have.property('access_token');
						var token = jwt.decode(res.body.access_token, secret);
						expect(token.type).to.equal('access');
						expect(token.id).to.equal(u2.id);
				})
				.end(done);
		})

		it('should fail if user is invalid', function(done){
			request(app)
				.post('/auth')
				.send({ username: 'xxx@xxx.com', password: 'bad', as: u.id })
				.expect(403)
				.expect(function(res){
						expect(res.body).to.have.property('errors');
						expect(res.body.errors[0]).to.equal(errors.InvalidUserId);
				})
				.end(done);
		})

	})


	describe("refresh_token", function(){

		var refresh_token, token;

		before(function(done){
			request(app)
				.post('/auth')
				.send({ username: u.id, password: u.password })
				.expect('Content-Type', /json/)
				.expect(200)
				.expect(function(res){
					expect(res.body).to.have.property('refresh_token');
					refresh_token = res.body.refresh_token;
					token = jwt.decode(res.body.refresh_token, secret);
				})
				.end(done)
		})

		it('should be an object', function(){
			expect(token).to.be.an.object;
		})

		it('should have type=refresh', function(){
			expect(token).to.have.property('type');
			expect(token.type).to.equal('refresh');
		})

		it('should contain a user uuid', function(){
			expect(token).to.have.property('id');
			expect(token.id).to.be.a.string;
			expect(token.id).to.match(/^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
		})

		it('should expire after 1 day', function(){
			var now = moment();
			expect(token).to.have.property('exp');
			expect(moment(token.exp).isAfter(now)).to.be.true;
			expect(moment(token.exp).isBefore(now.add(2, 'days'))).to.be.true;
		})

		it('should be exchangable for a fresh access_token and refresh_token', function(done){
			request(app)
				.post('/auth')
				.send({ token: refresh_token })
				.expect('Content-Type', /json/)
				.expect(function(res){
					var now = moment();
					expect(res.body).to.have.property('access_token');
					var access_token = jwt.decode(res.body.access_token, secret);
					expect(access_token).to.have.property('exp');
					expect(moment(access_token.exp).isAfter(now)).to.be.true;
					expect(access_token).to.have.property('id');
					expect(access_token.id).to.equal(token.id);
					expect(res.body).to.have.property('refresh_token');
					var refresh_token = jwt.decode(res.body.refresh_token, secret);
					expect(refresh_token).to.have.property('exp');
					expect(moment(refresh_token.exp).isAfter(now)).to.be.true;
					expect(refresh_token).to.have.property('id');
					expect(refresh_token.id).to.equal(token.id);
				})
				.end(done)
		})


	})

	describe("lockout", function(){

		var u;

		// create a user that we will lockout
		before(function(done){
			generateRandomUser().then(function(user){
				su = user;
				generateRandomUser().then(function(user){
					u = user;
					done();
				})
			})
		});

		it('should lockout logins if 10 failed attempts within 10 mins', function(done){
			var attempts = [];
			for(var i=0; i<10; i++){
				attempts.push(function(){
					return new Promise(function(resolve, reject){
						request(app)
						.post('/auth')
						.send({ username: u.id, password: 'badpassword' })
						.expect(function(res){
							expect(res.body).to.have.property('errors');
							expect(res.body.errors[0]).to.equal(errors.InvalidPassword);
						}).end(resolve)
					})
				})
			}
			attempts.reduce(function(p, attempt){
				return p.then(attempt)
			},Promise.resolve()).then(function(){
				return new Promise(function(resolve){
					request(app)
						.post('/auth')
						.send({ username: u.id, password: 'badpassword' })
						.expect(function(res){
							expect(res.body).to.have.property('errors');
							expect(res.body.errors[0]).to.equal(errors.LockedUserId);
						}).end(resolve)
				})
			}).then(function(){
				request(app)
					.post('/auth')
					.send({ username: u.id, password: u.password })
					.expect(function(res){
						expect(res.body).to.have.property('errors');
						expect(res.body.errors[0]).to.equal(errors.LockedUserId);
					})
					.end(done)
			})

		})

		it('should clear lockout after some time', function(done){
			app.passwd.failures.expire = {ms: 1}; // force early expiry
			app.passwd.failures.collectGarbage(); // to simulate time past
			request(app)
				.post('/auth')
				.send({ username: u.id, password: u.password })
				.expect(function(res){
					expect(res.body).to.not.have.property('errors');
					expect(res.body).to.have.property('access_token');
				})
				.end(done)
		})

	})


})
