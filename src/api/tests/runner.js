#!/usr/bin/env node

var Mocha = require('mocha'),
	fs = require('fs'),
	path = require('path');

// First, you need to instantiate a Mocha instance.
var mocha = new Mocha();

// Here is an example:
fs.readdirSync(__dirname).filter(function(file){
	if( file == 'runner.js' ){
		return false;
	}
	return file.substr(-3) === '.js';
}).forEach(function(file){
	// Use the method "addFile" to add the file to mocha
	mocha.addFile(
		path.join(__dirname, file)
	);
});

// enable debug
process.env.DEBUG = "true";

// Now, you can run the tests.
mocha.reporter('list').bail(true).run(function(failures){
	//process.on('exit', function () {
		process.exit(failures);
	//});
});
