import * as actions from "./actions";
import * as schema from "./schema";

arla.configure({
  // verbosity sets which level of logs will be output.
  // possible options are: NONE=0, DEBUG=1, INFO=2, LOG=3, WARN=2, ERROR=1
  verbosity: console.DEBUG,
  // engine tells arla which queryengine to use.
  // note: this does nothing now, it's always postgres.
  engine: 'postgres',
  // actions declares the mutation functions that are exposed
  actions: actions,
  // schema is an Object that declares the struture of your data
  // and how queries should be built.
  schema: schema,

});
