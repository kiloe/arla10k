import Kefir from 'kefir';
import {v4} from 'node-uuid';

// import action names
const ACTIONS = ['__ACTIONS_LIST__'];

// polyfills
import 'babelify/polyfill';
import 'whatwg-fetch';
var fetch = window.fetch;

// posible datastore states
export const UNAUTHENTICATED = Symbol();
export const AUTHENTICATING = Symbol();
export const REGISTERING = Symbol();
export const AUTHENTICATED = Symbol();

function include(klass, name) {
	let queries = klass.queries ?
		klass.queries(klass.queryParams || {}, include) :
		{};
	if( !queries[name] ){
		throw Error(`failed to include subquery ${name} from ${klass.name}: does not exist`);
	}
	return queries[name];
}

//TODO: no need for difference between getQuery / include semantics really
function getQuery(klass, name) {
	return klass.query(klass.queryParams || {}, include);
}

// A thin wrapper around a stream solely to make it easier to
// remove the stream listeners by calling destroy() within componentWillUnmount
class Stream {
	constructor(datastore, stream){
		this.datastore = datastore;
		this.valueCallbacks = [];
		this.errorCallbacks = [];
		this.stream = stream;
	}

	onValue(callback){
		this.valueCallbacks.push(callback);
		this.stream.onValue(callback);
		return this;
	}

	onError(callback){
		this.errorCallbacks.push(callback);
		this.stream.onError(callback);
		return this;
	}

	destroy(){
		this.valueCallbacks.forEach(cb => this.results.offValue(cb));
		this.errorCallbacks.forEach(cb => this.results.offError(cb));
	}
}

// QueryStream listens for mutations to the datastore and
// then makes a request for the query obtained by klass.query()
class QueryStream extends Stream {
	constructor(datastore, klass){
		let stream = datastore.updates.flatMapLatest( () => {
			return Kefir.fromPromise(datastore.query(klass));
		}).toProperty();
		super(datastore, stream);
	}
}

class Datastore {

	constructor(url){
		this.url = url;
		// load tokens from localstorage
		this.tokens = JSON.parse(localStorage.getItem('tokens'));
		// is datastore online
		this._state = this.tokens ?
			AUTHENTICATED : UNAUTHENTICATED;
		this._stateStream = Kefir.emitter();
		this.state = this._stateStream.skipDuplicates().toProperty(this._state);
		// stream of successful actions executed
		this.actionRequests = Kefir.emitter();
		// create methods for each listed action from actions.js
		ACTIONS.forEach( k => {
			if( this[k] ){
				throw Error(`invalid action name: ${k}`)
			}
			this[k] = (...args) => {
				this.actionRequests.emit({
					name: k,
					args: args
				})
			}
		})
		// drain the actionRequest queue
		this.actionResponses = this.actionRequests.flatMap( ({name,args}) => {
			return Kefir.fromPromise(this._exec(name, ...args))
		})
		// stream of notifications that queries should update
		this.updates = Kefir.merge([
			Kefir.interval(30000, 1).filter( v => this._state == AUTHENTICATED),
			this.actionResponses.map( r => 1 ),
			this.state.filter(s => s == AUTHENTICATED).map( s => 1 )
		]).toProperty();
	}

	// generate a new uuid
	uuid() {
		return v4();
	}

	userId() {
		return this.tokens && this.tokens.id;
	}

	_post(url, o, headers){
		return fetch(url, {
			method: 'post',
			body: typeof o == 'string' ?
				o : JSON.stringify(o),
			headers: Object.assign({
				'Accept': 'application/json',
				'Content-Type': 'application/json'
			}, headers)
		}).then( res => {
			switch(res.status){
				case 200:
					return res.json()
				case 400:
					return res.json().then(o => {
						console.log('datastore.post received err:', o.errors);
						return Promise.reject(o.errors.join('\n'))
					})
				case 403:
					this.tokens = null;
					localStorage.removeItem('tokens');
					this.setState(UNAUTHENTICATED);
					return res.json().then(o => {
						console.log('datastore.post got 403 response');
						return Promise.reject(o.errors.join('\n'))
					})
					return Promise.reject('invalid password');
				default:
					console.log('datastore.post received unexpected error response:', res.status, res.json());
					return Promise.reject(res.statusText)
			}
		})
	}

	query(klass){
		if( this._state != AUTHENTICATED ){
			return Promise.reject(Error('cannot query datastore: not ready'))
		}
		return this._post('/query', getQuery(klass), {
			'Authorization': `bearer ${this.tokens.access_token}`,
			'Content-Type': 'text/plain'
		})
	}

	queryStream(klass){
		return new QueryStream(this, klass);
	}

	_exec(name, ...args){
		if( this._state != AUTHENTICATED ){
			return Promise.reject(Error('cannot exec datastore action: not ready'))
		}
		return this._post('/exec', {
			name: name,
			args: args
		}, {
			'Authorization': `bearer ${this.tokens.access_token}`,
		})
	}

	logout(){
		localStorage.removeItem('tokens');
		this.setState(UNAUTHENTICATED);
	}

	login(username, password){
		this.setState(AUTHENTICATING)
		return new Promise( (resolve, reject) => {
			this._post('/auth',{
				username: username,
				password: password
			}).then(tokens => {
				this.tokens = tokens;
				localStorage.setItem('tokens', JSON.stringify(tokens));
				this.setState(AUTHENTICATED);
				resolve(tokens);
			}).catch(errs => {
				this.tokens = null;
				this.setState(UNAUTHENTICATED);
				reject(errs);
			})
		})
	}

	register(profile){
		this.setState(REGISTERING);
		return new Promise( (resolve, reject) => {
			this._post('/register',profile).then(tokens => {
				this.tokens = tokens;
				localStorage.setItem('tokens', JSON.stringify(tokens));
				this.setState(AUTHENTICATED);
				resolve(tokens);
			}).catch(errs => {
				this.tokens = null;
				this.setState(UNAUTHENTICATED);
				reject(errs);
			})
		})
	}

	setState(state){
		this._state = state;
		this._stateStream.emit(state);
	}

	stateStream(){
		return new Stream(this, this.state);
	}

}

export var datastore = Object.assign(new Datastore('/'), {
	UNAUTHENTICATED,
	AUTHENTICATING,
	REGISTERING,
	AUTHENTICATED,
})
export default datastore;
