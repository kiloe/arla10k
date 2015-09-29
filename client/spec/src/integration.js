import {createClient} from '../../dist';

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
        .on('error', function(){});
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
        }).catch(done.fail);
      });

      it('should emit authenticated after registration (for bob)', function(done){
        client.register(bob).then(function(){
          expect(onAuthenticated).toHaveBeenCalled();
          done();
        }).catch(done.fail);
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

    beforeEach(function(done){
      client = createClient(cfg)
        .on('authenticated', done)
        .on('error', function(){})
        .connect(bob);
    })

    it('should be able to fetch me(){username}', function(done){
      client.query(`me(){username}`).then(function(data){
        expect(data).toEqual({
          me: {username: 'bob'}
        })
        done();
      }).catch(done.fail);
    });

    it('should be able to exec addEmailAddress mutation for bob', function(done){
      client.exec("addEmailAddress", "d79202c1-ce9f-4dfa-9a33-7b6387b49523", "bob@bob.com").then(function(ok){
        expect(ok).toBe(true);
        done();
      }).catch(done.fail);
    })

    it('should NOT be able to exec addEmailAddress mutation for bob (already exists)', function(done){
      client.exec("addEmailAddress", "a351370c-096e-49d1-9098-6ffc684fa287", "bob@bob.com").then(function(ok){
        expect(ok).not.toBe(true);
        done.fail('expected addEmailAddress to fail (on this attempt)');
      }).catch(function(err){
        done();
      });
    })

    it('should expose updateEmailAddress as a func on client', function(done){
      client.updateEmailAddress("bob@bob.com", "bob@gmail.com").then(function(ok){
        expect(ok).toBe(true);
        done();
      }).catch(done.fail);
    })

    it('should be able to fetch email address', function(done){
      client.query(`
        me(){
          email:
          email_addresses.pluck(addr).first()
        }
      `).then(function(data){
        expect(data).toEqual({
          me: {email: 'bob@gmail.com'}
        })
        done();
      }).catch(done.fail);
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
        .on('error', done.fail)
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
        .on('error', done.fail)
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
        .on('error', done.fail)
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

    describe('refresh', function(){

      it('should cause polling query to poll imeddiately', function(done){
        let query = client.prepare(`members(){username}`)
          .on('data', function(data){
            expect(data).toEqual({
              members: [{username: "alice"},{username: "bob"}]
            })
            done();
          })
          .on('error', done.fail)
          .poll(50000);
        client.refresh();
      })
    })

  })

  describe('query polling', function(){
    let client;
    let query;
    let i;
    let ql = `me(){username}`;

    let doneAfterCalled = function(n, done){
      return function(data){
        expect(data).toEqual({me: {username: "bob"}});
        i++;
        if( i == n ){
            query.stop().then(done);
        }
        if( i > n ){
          done.fail('doneAfterCalled called '+i+' times (expected '+n+')');
        }
      }
    }

    beforeEach(function(){
      i = 0;
      client = createClient(cfg);
    })

    it('should periodically fetch data', function(done){
      query = client.on('error', done.fail).connect(bob)
        .prepare(ql)
        .on('data', doneAfterCalled(3, done))
        .on('error', done.fail)
        .poll(10);
    })

    it('should only start after authenticated', function(done){
      query = client.on('error', done.fail)
        .prepare(ql)
        .on('data', doneAfterCalled(3, done))
        .on('error', done.fail)
        .poll(10);
      setTimeout(function(){
        expect(i).toEqual(0);
        client.connect(bob);
      },10);
    })

    it('should pause if becomes unauthenticated', function(done){
      query = client.connect(bob)
        .prepare(ql)
        .on('data', doneAfterCalled(10, done))
        .on('error', done.fail)
        .poll(10);
      setTimeout(function(){
        client.deauthenticate();
      },10);
      setTimeout(function(){
        client.authenticate(bob);
      },30);
    })

  })

});
