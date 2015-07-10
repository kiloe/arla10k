import * as actions from "./actions";
import * as schema from "./schema";

arla.configure({
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
      select id from member
      where username = $1
      and password = crypt($2, password)
    `, username, password];
  },
  // the register function returns the mutation-action action that will
  // be executed to register a new user. Unlike other mutations this
  // one should also build a temporary token
  register({username, password}){
    return {
      Name: "registerMember",
      Args:[{
        username: username,
        password: pgcrypto.crypt(password)
      }]
    }
  }

});
