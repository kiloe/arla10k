import {EventEmitter} from 'events';

export class Query extends EventEmitter {

  constructor({client, builder}){
    super();
    this.client = client;
    this.builder = builder;
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
