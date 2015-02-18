#!/usr/bin/env node

var app = require('./app')();

app.start().then(function(app){
	console.log('Started server on ' + app.port);
}).catch(function(err){
	console.error(err);
	process.exit(1);
});
