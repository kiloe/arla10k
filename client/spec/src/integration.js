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
        done.fail(err.error || err)
      });
    });

    it('should emit authenticated after login', function(done){
      client.authenticate(bob).then(function(){
        expect(onUnauthenticated).toHaveBeenCalled();
        done();
      }).catch(function(err){
        done.fail(err.error || err)
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
        done.fail(err.error || err)
      });
    });

    it('should be able to exec addEmailAddress mutation for bob', function(done){
      client.exec("addEmailAddress", "bob@bob.com").then(function(ok){
        expect(ok).toBe(true);
        done();
      }).catch(function(err){
        done.fail(err.error || err)
      });
    })

    it('should NOT be able to exec addEmailAddress mutation for bob (already exists)', function(done){
      client.exec("addEmailAddress", "bob@bob.com").then(function(ok){
        expect(ok).not.toBe(true);
        done.fail('expected addEmailAddress to fail (on this attempt)');
      }).catch(function(err){
        done();
      });
    })

    it('should be able to fetch email address', function(done){
      client.query(`
        me(){
          email:
          email_addresses.pluck(addr).first()
        }
      `).then(function(data){
        expect(data).toEqual({
          me: {email: 'bob@bob.com'}
        })
        done();
      }).catch(function(err){
        done.fail(err.error || err)
      });
    });
  })



});
