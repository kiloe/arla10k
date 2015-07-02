
DO $$
  try{
    plv8.arla.init();
  } catch (e) {
    plv8.elog(ERROR, e.stack || e.message || e.toString());
  }
$$ LANGUAGE plv8;
