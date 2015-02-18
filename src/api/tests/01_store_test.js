var tmp = require('tmp');
var store = require('../store');
var Promise = require('es6-promise').Promise;
var expect = require('chai').expect;
var path = require('path');
var moment = require('moment');

describe('log store', function(){

	var tmpDir;

	before(function(done){
		tmp.setGracefulCleanup();
		tmp.dir({unsafeCleanup: true, prefix: '.testtmp'}, function(err, path){
			expect(err).to.be.null;
			tmpDir = path;
			done();
		})
	})

	it('should commit json objects and read them back as a stream', function(done){
		var a = {thing: 100};
		var b = {thing: 200};
		var c = {thing: 300};
		var log;
		Promise.resolve().then(function(){
			return store.open(path.join(tmpDir, 't1.wbl'))
		}).then(function(s){
			log = s;
			return log.put(a);
		}).then(function(id){
			expect(id).to.be.a.number;
			return log.put(b);
		}).then(function(id){
			expect(id).to.be.a.number;
			return log.put(c);
		}).then(function(id){
			expect(id).to.be.a.number;
			return log.stream()
		}).then(function(stream){
			var i = 0;
			stream.onValue(function(o){
				i++;
				switch(i){
				case 1: expect(o.value).to.deep.equal(a); break;
				case 2: expect(o.value).to.deep.equal(b); break;
				case 3: expect(o.value).to.deep.equal(c); break;
				default: throw Error("There should only be 3 values in the stream");
				}
			});
			stream.onEnd(done);
		}).catch(done);
	});

	it('should be possible to stream after a given id', function(done){
		var a = {x: {y: 'z'}};
		var b = {a: {b: 'c'}};
		var c = {x: {y: 3}};
		var from;
		var log;
		Promise.resolve().then(function(){
			return store.open(path.join(tmpDir, 't2.wbl'))
		}).then(function(s){
			log = s;
			return log.put(a);
		}).then(function(id){
			return log.put(b);
		}).then(function(id){
			from = id;
			return log.put(c);
		}).then(function(id){
			return log.stream(from);
		}).then(function(stream){
			var i = 0;
			stream.onValue(function(o){
				i++;
				switch(i){
				case 1: expect(o.value).to.deep.equal(b); break;
				case 2: expect(o.value).to.deep.equal(c); break;
				default: throw Error("There should only be 2 values in the stream");
				}
			});
			stream.onEnd(done);
		}).catch(done);
	});

	it('should pass the serial id in the stream', function(done){
		var id1, id2, id3;
		var log;
		Promise.resolve().then(function(){
			return store.open(path.join(tmpDir, 't3.wbl'))
		}).then(function(s){
			log = s;
			return log.put({t:1});
		}).then(function(id){
			id1 = id;
			return log.put({t:2});
		}).then(function(id){
			id2 = id;
			return log.put({t:3});
		}).then(function(id){
			id3 = id;
			return log.stream();
		}).then(function(stream){
			var i = 0;
			stream.onValue(function(o){
				expect(o.id).to.be.a('number');
				i++;
				switch(i){
				case 1: expect(o.id).to.equal(id1); break;
				case 2: expect(o.id).to.equal(id2); break;
				case 3: expect(o.id).to.equal(id3); break;
				default: throw Error("There should only be 3 values in the stream");
				}
			});
			stream.onEnd(done);
		}).catch(done);
	});

	it('should return the date in the stream', function(done){
		var d1, d2, d3;
		var log;
		Promise.resolve().then(function(){
			return store.open(path.join(tmpDir, 't3.wbl'))
		}).then(function(s){
			log = s;
			return log.put({t:1});
		}).then(function(id){
			return log.put({t:2});
		}).then(function(id){
			return log.put({t:3});
		}).then(function(id){
			return log.stream();
		}).then(function(stream){
			var i = 0;
			stream.onValue(function(o){
				expect(o.at).to.have.property('isBefore');
				expect(o.at.isBefore(moment())).to.equal(true);
			});
			stream.onEnd(done);
		}).catch(done);
	});


})
