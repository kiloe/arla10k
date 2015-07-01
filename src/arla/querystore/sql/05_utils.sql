
-- since(x) === age(now(), x)
CREATE OR REPLACE FUNCTION since(t timestamptz) RETURNS interval AS $$
	select age(now(), t);
$$ LANGUAGE "sql" VOLATILE;

-- until(x) === age(x, now())
CREATE OR REPLACE FUNCTION until(t timestamptz) RETURNS interval AS $$
	select age(t, now());
$$ LANGUAGE "sql" VOLATILE;
