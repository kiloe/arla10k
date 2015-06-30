import * as actions from "./actions";
import * as schema from "./schema";

arla.configure({
  engine: 'postgres',
  actions: actions,
  schema: schema
});
