import {EventEmitter} from 'events';

export class Query extends EventEmitter {

  constructor({client, builder}){
    super();
    this.client = client;
    this.builder = builder;
    this.pollInterval = 30000;
    this.on('error', function(err){
      if( EventEmitter.listenerCount(this, 'error') < 2 ){
        console.error(err);
      }
    });
  }

  // run executes the query exactly once.
  // returns a promise with the query data
  run(){
    let args = this._getQuery();
    return this.client.query(...args).then(data => {
      this.emit('data', data);
      return data;
    }).catch(ex => {
      this.emit('error', ex);
      return Promise.reject(ex);
    });
  }

  // poll executes run() continuously in intervals of ms
  // until stop() is called;
  // returns the Query for chaining
  poll(ms){
    this._poll(ms);
    return this;
  }

  // _poll executes run() continuously in intervals of ms
  // until stop() is called;
  _poll(ms){
    if( !ms ){
      ms = this.pollInterval;
    }
    if( this.polling ){
      throw new Error('query is already polling');
    }
    this.polling = this.run().catch( () => null).then( () => {
      if( !this.polling ){
        return;
      }
      return new Promise( (resolve, reject) => {
        setTimeout( () => {
          this.polling = false;
          resolve(this._poll(ms))
        }, ms);
      })
    })
    return this.polling;
  }

  // stop halts the query polling.
  // returns a promise that resolves when no more requests are active.
  stop(){
    let last = this.polling;
    this.polling = false;
    return last ? last.then( () => true ) : Promise.resolve(true);
  }

  // getQuery converts the builder into args suitable for Client#query
  _getQuery(){
    let builder = this.builder;
    if( !builder ){
      throw new Error('query.builder cannot be undefined/null/false');
    }
    if( !Array.isArray(builder) ){
      throw new Error('query.builder should be an Array');
    }
    if( builder.length == 0 ){
      throw new Error('query.builder should be an Array with at least one element');
    }
    if( builder.length == 1 ){
      builder = builder[0];
    }
    if( !builder ){
      throw new Error('query.builder contained an invalid element');
    }
    switch( typeof builder ){
      case 'string':
        return [builder];
      case 'function':
        let args = builder();
        if( !args ){
          throw new Error('query builder function must return an Array of query args');
        }
        if( !Array.isArray(args) ){
          args = [args];
        }
        return args;
      case 'object':
        if( Array.isArray(builder) ){
          return builder;
        }
      default:
        throw new Error('query builder must be either an Array of args or a function that returns an Array of args');
    }
  }

}
