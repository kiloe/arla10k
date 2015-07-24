import {EventEmitter} from 'events';

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

  constructor({ url = '/', token = null }){
    super();
    // let _emit = this.emit;
    // this.emit = (...args) => {
    //   setTimeout(_emit.bind(this, ...args),0)
    // }
    this.url = absurl(url);
    this.token = token;
    this.on('error', function(err){
      if( EventEmitter.listenerCount(this, 'error') < 2 ){
        log(err);
      }
    });
  }

  connect(credentials){
    if( credentials ){
      this.authenticate(credentials);
    } else {
      this.setToken(this.token);
    }
    return this;
  }

  disconnect(){
    this.info = null;
  }

  // configure continuously polls for /info until it gets a response.
  configure(){
    if( this.info ){
      return Promise.resolve(this.info);
    }
    return this._post('info').then(res => {
      this.info = res.data;
      return this.info
    }).catch(ex => {
      // If error return a promise to try again
      this.emit('error', {error: 'connection failure... retrying', reason: ex});
      return new Promise( (resolve) => {
        setTimeout( () => {
          resolve(this.configure())
        }, 1000)
      })
    })
  }

  // authenticate
  authenticate(values){
    return this.post('authenticate', values)
      .then(this.maybeAuthenticate.bind(this))
  }

  // deauthenticate removes the token and disables the client
  // until authentication.
  deauthenticate(){
    this.setToken(null);
  }

  // setToken assigns an authentication token and triggers an event
  setToken(token){
    this.token = token;
    if( this.token ){
      this.emit(AUTHENTICATED);
    } else {
      this.emit(UNAUTHENTICATED);
    }
  }

  // register sends a registration request to the server and returns a
  // promise.
  register(values){
    return this.post('register', values)
      .then(this.maybeAuthenticate.bind(this));
  }

  // query sends a query request to the server and returns a promise
  query(q, ...args){
    return this.post('query', {
      query: q,
      args: args
    }).then( res => res.data )
  }

  // exec sends an exec request to the server and returns a promise
  exec(name, ...args){
    return this.post('exec', {
      name: name,
      args: args
    }).then( res => !!res.data.success)
  }

  // post returns a Promise for an API request.
  // The promise will either return the data or a rejection.
  post(...args){
    return this.configure().then( info => {
      return this._post(...args)
    })
  }

  _post(url, values){
    let opts = {
      method: 'post',
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json'
      }
    }
    if( this.token ){
      opts.headers['Authorization'] = `bearer ${this.token}`;
    }
    if( values ){
      opts.body = JSON.stringify(values);
    }
    return fetch(`${this.url}${url}`, opts)
      .then(this.normalizeResponse.bind(this))
      .then(this.maybeDeauthenticate.bind(this))
      .then(this.maybeReject.bind(this));
  }

  // normalizeResponse converts any non-json response to a json error
  // Parses the JSON response and places it either in res.error or res.data.
  normalizeResponse(res){
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
  maybeDeauthenticate(res){
    if( res.status == 403 ){
      this.deauthenticate();
    }
    return res;
  }

  // maybeAuthenticate checks the data for an access_token and
  // triggers the client to store it before returning the response unchange
  maybeAuthenticate(res){
    if( res.data && res.data.access_token ){
      this.setToken(res.data.access_token);
    }
    return res;
  }

  // maybeReject converts error responses to rejections or returns
  // the response unchanged
  maybeReject(res){
    if( res.error ){
      this.emit('error', res.error);
      return Promise.reject(res.error);
    }
    return res;
  }

}

export function createClient(cfg = {url: '/'}){
  return new Client(cfg)
}
