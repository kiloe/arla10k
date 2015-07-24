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

  var alice = {
    id:'c47ebce0-1e31-45be-9b84-aad235409305',
    username:'alice',
    password:'ali&&&%%£__-'
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
    })

    describe('without credentials', function(){

      beforeEach(function(done){
        client
          .on('unauthenticated', done)
          .connect();
      })

      it('should emit unauthenticated imediately', function(){
        expect(onUnauthenticated).toHaveBeenCalled();
      });

    });

    describe('with credentials', function(){

      beforeEach(function(done){
        client
          .on('unauthenticated', done)
          .on('authenticated', done)
          .connect(bob);
      })

      it('should emit unauthenticated after failed login', function(){
        expect(onUnauthenticated).toHaveBeenCalled();
      });

      it('should emit authenticated after registration (for alice)', function(done){
        client.register(alice).then(function(){
          expect(onAuthenticated).toHaveBeenCalled();
          done();
        }).catch(function(err){
          done.fail(err.error || err)
        });
      });

      it('should emit authenticated after registration (for bob)', function(done){
        client.register(bob).then(function(){
          expect(onAuthenticated).toHaveBeenCalled();
          done();
        }).catch(function(err){
          done.fail(err.error || err)
        });
      });

      it('should emit authenticated after successful connect', function(){
        expect(onAuthenticated).toHaveBeenCalled();
      });

    })

    describe('with token', function(){

      beforeEach(function(done){
        client
          .on('authenticated', done)
          .connect('abc123');
      })

      it('should emit authenticated imediately', function(){
        expect(onAuthenticated).toHaveBeenCalled();
      });

    })

  })


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

  describe('prepare', function(){
    let client;
    let onError;

    beforeEach(function(done){
      onError = jasmine.createSpy('onError');
      client = createClient(cfg)
        .on('authenticated', done)
        .on('error', onError)
        .connect(bob);
      expect(onError).not.toHaveBeenCalled();
    })

    it('should create a Query using a simple string', function(done){
      client.prepare(`members(){username}`)
        .on('data', function(data){
          expect(data).toEqual({
            members: [{username: "alice"},{username: "bob"}]
          })
          done();
        })
        .on('error', function(err){
          done.fail(err.error || err)
        })
        .run();
    })

    it('should create a Query using a simple string + args', function(done){
      client.prepare([`members().filter(id = $1){username}`,alice.id])
        .on('data', function(data){
          expect(data).toEqual({
            members: [{username: "alice"}]
          })
          done();
        })
        .on('error', function(err){
          done.fail(err.error || err)
        })
        .run();
    })

    it('should build query with a func', function(done){
      let queryBuilder = function(){
        return `members(){username}`
      };
      client.prepare(queryBuilder)
        .on('data', function(data){
          expect(data).toEqual({
            members: [{username: "alice"},{username: "bob"}]
          })
          done();
        })
        .on('error', function(err){
          done.fail(err.error || err)
        })
        .run();
    })

    it('should call builder before each run()', function(done){
      let queryBuilder = jasmine.createSpy('builder', function(){
        return `me(){username}`;
      }).and.callThrough();
      let query = client.prepare(queryBuilder);
      let qs = [];
      qs.push(query.run());
      qs.push(query.run());
      Promise.all(qs).then(function(){
        expect(queryBuilder.calls.count()).toEqual(2);
        done();
      });
    })

  })

});
