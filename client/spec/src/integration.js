import {createClient} from '../../dist/client';

describe('client', function(){

  var cfg = {
    url: 'http://localhost:3000/'
  };

  var bob = {
    id:'97f5442e-67a2-4f2b-8c12-82f5663daf43',
    username:'bob',
    password:'bobzpasswerd'
  };

  describe('authentication', function(){
    let client;
    let onAuthenticated;
    let onUnauthenticated;

    beforeEach(function(){
      onAuthenticated = jasmine.createSpy('onAuthenticated');
      onUnauthenticated = jasmine.createSpy('onUnauthenticated');
      client = createClient(cfg)
        .on('authenticated', onAuthenticated)
        .on('unauthenticated', onUnauthenticated)
        .connect();
    })

    it('should emit unauthenticated imediately when created without token', function(){
      expect(onUnauthenticated).toHaveBeenCalled();
    });

    it('should emit unauthenticated after failed login', function(done){
      client.authenticate(bob).then(function(){
        done.fail(`expected authenticated to fail since user should not exist`);
      }).catch(function(err){
        expect(onUnauthenticated).toHaveBeenCalled();
        done();
      });
    });

    it('should emit authenticated after registration', function(done){
      client.register(bob).then(function(){
        expect(onAuthenticated).toHaveBeenCalled();
        done();
      }).catch(function(err){
        done.fail(`failed to register: ${err.error || err}`)
      });
    });

    it('should emit authenticated after login', function(done){
      client.authenticate(bob).then(function(){
        expect(onUnauthenticated).toHaveBeenCalled();
        done();
      }).catch(function(err){
        done.fail(`failed to authenticate: ${err.error || err}`)
      });
    });
  })

  describe('with stored token', function(){
    it('should emit authenticated immediately', function(done){
      createClient({token: 'abc123'})
        .on('authenticated', done)
        .connect();
    });
  });

  describe('query', function(){
    let client;
    let onError;

    beforeEach(function(done){
      onError = jasmine.createSpy('onError');
      client = createClient(cfg)
        .on('authenticated', done)
        .on('error', onError)
        .connect(bob);
    })

    it('should be able to fetch me(){username}', function(done){
      client.query(`me(){username}`).then(function(data){
        expect(data).toEqual({
          me: {username: 'bob'}
        })
        done();
      }).catch(function(err){
        done.fail(`query failed: ${err.error || err}`)
      });
    });
  })



});
