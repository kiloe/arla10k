import * as actions from "./actions";
import * as schema from "./schema";

arla.configure({
  // verbosity sets which level of logs will be output.
  // possible options are: DEBUG=1, INFO=2, LOG=3, WARN=2, ERROR=1
  logLevel: console.INFO,
  // engine tells arla which queryengine to use.
  // note: this does nothing now, it's always postgres.
  engine: 'postgres',
  // actions declares the mutation functions that are exposed
  actions: actions,
  // schema is an Object that declares the struture of your data
  // and how queries should be built.
  schema: schema,
  // the authenticate function accepts user credentials and returns
  // the query that will return the values that will be used as the
  // context/claims/session for future requests
  authenticate({username, password}){
    return [`
      select true as admin
      where $1::text == 'admin'
      and $2::text == 'secret'
    `, username, password];
  },
  // the register function returns the mutation-action action that will
  // be executed to register a new user. Unlike other mutations this
  // one should also build a temporary token
  register({username, password}){
    throw 'regitrations are closed!';
  }

});
