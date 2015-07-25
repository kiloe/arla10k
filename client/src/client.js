import {EventEmitter} from 'events';
import {Query} from './query';

// polyfills
import 'babelify/polyfill';
import 'isomorphic-fetch';

// log writes a console log message
function log(...args){
  if( typeof console != 'undefined' ){
    console.log(...args);
  }
}

// absurl converts relative url to absolute url.
// if window object is available it uses the location
// else it assumes localhost.
function absurl(path){
  if( path.startsWith('http') ){
    return path;
  }
  let host = 'http://localhost';
  if( typeof window != 'undefined' ){
    host = `${window.location.protocol}//${window.location.host}`;
  }
  return `${host}${path}`
}

// Possible events emitted by client
const UNAUTHENTICATED = 'unauthenticated';
const AUTHENTICATED = 'authenticated';

// Client is a connection to arla.
export class Client extends EventEmitter {

  constructor({ url = '/' }){
    super();
    this.url = absurl(url);
    this.state = UNAUTHENTICATED;
    this.on('error', function(err){
      if( EventEmitter.listenerCount(this, 'error') < 2 ){
        log(err);
      }
    });
    this.on(AUTHENTICATED, e => {
      this._setState(AUTHENTICATED)
    })
    this.on(UNAUTHENTICATED, e => {
      this._setState(UNAUTHENTICATED);
    })
  }

  // connect sets up the authentication details for future
  // requests with the client. after connect is called either
  // an 'authenticated' or 'unauthenticated' event will be emitted.
  connect(credentials){
    // connect without args starts unauthenticated
    if( !credentials ){
      this._setToken(null);
      return this;
    }
    // connect with a string arg sets the token and starts authenticated
    if( typeof credentials == 'string'){ // is a token
      this._setToken(credentials);
      return this;
    }
    // any other type of arg is assumed to be login details
    this.authenticate(credentials);
    return this;
  }

  // disconnect clears the cached info, cancels any active queries
  // and deauthenticates the current user.
  disconnect(){
    this.info = null;
    this.deauthenticate();
    return this;
  }

  // register sends a registration request to the server and returns a
  // promise.
  register(values){
    return this._post('register', values)
      .then(res => this.authenticate(values))
  }

  // query sends a query request to the server and returns a promise
  query(q, ...args){
    return this._query(q, args);
  }

  _query(q, args, opts = {}){
    return this._post('query', {
      query: q,
      args: args
    },opts).then( res => res.data )
  }

  // exec sends an exec request to the server and returns a promise
  exec(name, ...args){
    return this._post('exec', {
      name: name,
      args: args
    }).then( res => !!res.data.success)
  }

  // prepare creates a prepared Query.
  // a Query can be executed mutliple times, has the ability to
  // regenerate it's AQL via a builder function and has methods to
  // make polling/refreshing queries easier.
  prepare(...args){
    return new Query({client: this, builder: args});
  }


  // configure continuously polls for /info until it gets a response.
  _configure(){
    if( this.info ){
      return Promise.resolve(this.info);
    }
    return this._req('get', 'info').then(res => {
      this.info = res.data;
      return this.info
    }).catch(ex => {
      // If error return a promise to try again
      this.emit('error', {error: 'connection failure... retrying', reason: ex});
      return new Promise( (resolve) => {
        setTimeout( () => {
          resolve(this._configure())
        }, 1000)
      })
    })
  }

  // authenticate
  authenticate(values){
    return this._post('authenticate', values)
      .then(res => res.data.access_token)
      .catch(ex => null)
      .then(token => this._setToken(token));
  }

  // deauthenticate removes the token and disables the client
  // until authentication.
  deauthenticate(){
    this._setToken(null);
  }

  // setToken assigns an authentication token and triggers an event
  _setToken(token){
    this.token = token;
    if( this.token ){
      this.emit(AUTHENTICATED);
    } else {
      this.emit(UNAUTHENTICATED);
    }
    return token;
  }

  // setState assign the current state
  // if new state differs from old state a 'change' event is emitted
  // with the new state
  _setState(state){
    let old = this.state;
    this.state = state;
    if( old != state ){
      this.emit('change', state);
    }
    return state;
  }

  // post returns a Promise for an API request.
  // The promise will either return the data or a rejection.
  _post(url, values, opts = {}){
    Object.assign(opts, {token: this.token});
    return this._configure().then( info => {
      return this._req('post', url, values, opts)
    })
  }

  _req(method, url, values, opts){
    if( !opts ){
      opts = {};
    }
    let req = {
      method: method || 'post',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      }
    }
    if( opts.token ){
      req.headers['Authorization'] = `bearer ${opts.token}`;
    }
    if( values ){
      req.body = JSON.stringify(values);
    }
    if( !opts.emitter ){
      opts.emitter = this;
    }
    return fetch(`${this.url}${url}`, req)
      .then(this._normalizeResponse.bind(this))
      .then(this._maybeDeauthenticate.bind(this))
      .then(this._maybeReject.bind(this, opts.emitter));
  }

  // normalizeResponse converts any non-json response to a json error
  // Parses the JSON response and places it either in res.error or res.data.
  _normalizeResponse(res){
    let ct = res.headers.get('Content-Type');
    if( ct != 'application/json' ){
      return res.text().then( txt => {
        res.error = {error: `unexpected response type ${ct}`, body:txt}
        return res;
      }).catch( ex => {
        res.error = {error: `unexpected response type ${ct}: ${ex}`};
        return res;
      })
    }
    return res.json().then( json => {
      if( res.status == 200 ){
        res.data = json;
      } else {
        res.error = json;
      }
      return res;
    }).catch( ex => {
      res.error = {error: 'failed to parse json response'};
      return res;
    })
  }

  // maybeDeauthenticate checks for 403 responses and triggers
  // the client to deauth before passing on the response unchanged
  _maybeDeauthenticate(res){
    if( res.status == 403 ){
      this.deauthenticate();
    }
    return res;
  }

  // maybeReject converts error responses to rejections or returns
  // the response unchanged
  _maybeReject(emitter, res){
    if( res.error ){
      let err = res.error.error || res.error;
      emitter.emit('error', err);
      return Promise.reject(err);
    }
    return res;
  }

}

export function createClient(cfg = {url: '/'}){
  return new Client(cfg)
}
