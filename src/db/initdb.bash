#!/bin/bash

# this script should be run from the arla lib dir
# usually found somewhere like /var/lib/arla

LIB=$PWD
LOG=$LIB/init.stderr
OUT=$LIB/init.stdout
SQL=$LIB/template.compiled.sql

# try will attempt to execute the command given as it's argument in a subshell
# if the subcommand returns a non-zero exit code then the commands error output
# will be parsed for useful context which will be output before exiting.
function try() {
	echo '' > $LOG
	( "$@" ) 2> $LOG > $OUT
	if [ $? -ne 0 ]; then
		if [ -e $SQL ]; then
			line=$(grep 'plv8_init:' < $LOG | head -1 | sed -e 's/.*plv8_init://' | cut -d: -f1)
			if [ -z "$line" ]; then
				line=$(grep 'plv8_init() LINE' < $LOG | cut -d' ' -f5 | tr -d ':')
				line=$(($line + 1)) # offset
			fi
			if [ ! -z "$line" ]; then
				line=$(($line + 8)) # offset from start of template.sql to init func
				echo
				echo "---------------"
				grep --color -m1 -A1 '^ERROR:.*$' < $LOG
				echo
				cat -n $SQL | grep -E --color -C6 "^\s+${line}\s+.*$"
				echo
				echo "---------------"
				echo
			else
				cat $OUT $LOG
				echo "could not determine line ${line}"
			fi
		else
			cat $OUT $LOG
		fi
		echo "FAILED during: $@"
		exit 1
	fi
	cat $OUT
}

echo "creating the database..."
try createdb $PGDATABASE

echo "compiling app sources..."
try browserify index.js -t [ /usr/local/lib/node_modules/babelify --modules common ] -o index.compiled.js

echo "installing extensions..."
try sed -e "/index.js/ {" -e "r index.compiled.js" -e "d" -e "}" template.sql > $SQL
try psql -1 -v ON_ERROR_STOP=1 < $SQL

echo "running schema migrations..."
try psql -c "SELECT arla_migrate();" -v ON_ERROR_STOP=1
