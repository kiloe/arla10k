var request = require('supertest'),
	expect = require('chai').expect,
	Promise = require('es6-promise').Promise,
	secret = 'testing',
	app = require('./helper/app'),
	uuid = require('node-uuid');

// query helper
function q(sql, args){
	return new Promise(function(resolve, reject){
		pg.connect(app.conString, function(err, client, done) {
			db.query(sql, args, function(err, result){
				if( err ){
					return reject(err);
				}
				resolve(result.rows);
			})
		})
	})
}

describe('sync', function(){

	var user1, user2;

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
				user1 = u;
				done();
			})
	})

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
				user2 = u;
				done();
			})
	})

	it('should sync query data after resetting', function(done){
		var originalCount;
		// destroy database
		app.query('select * from member').then(function(rows){
			originalCount = rows.length;
			expect(originalCount).to.be.greaterThan(0);
		}).then(function(){
			return app.query('select arla_destroy_data()')
		}).then(function(){
			return app.query('select * from member');
		}).then(function(rows){
			expect(rows.length).to.equal(0);
		}).then(function(){
			return app.sync();
		}).then(function(){
			return app.query('select * from member');
		}).then(function(rows){
			expect(rows.length).to.equal(originalCount);
		}).then(done).catch(function(err){
			done(err)
		})
	})
})
