
var SHOW_DEBUG = true;

function logger(level, ...msgs) {
	var msg = msgs.map(function(msg){
		if( typeof msg == 'object' ){
			msg = JSON.stringify(msg, null, 4);
		}
		return msg;
	}).join(' ');
	(msg || '').split(/\n/g).forEach(function(line){
		plv8.elog(level, line);
	})
}

export function log(...msg) {
	logger(NOTICE, ...msg)
}

export function debug(...msg) {
	if( SHOW_DEBUG ){
		log(...msg)
	}
}

export function warn(...msg) {
	logger(WARNING, ...msg)
}

export function error(...msg) {
	// don't use ERROR as this will halt execution
	logger(WARNING, ...msg)
}

export default {
		log,
		debug,
		warn,
		error
}
