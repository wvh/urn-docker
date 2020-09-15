#!/bin/bash
#
# -wvh- create database, user and schema; init tables; insert initial data (if any).
#

set -e

SQLFILES="database.sql initial.sql testdata.sql"

POSTGRES_USER=${POSTGRES_USER-postgres}
POSTGRES_DB=${POSTGRES_DB-postgres}

# superuser variables
echo "POSTGRES_DB = $POSTGRES_DB"
echo "POSTGRES_USER = $POSTGRES_USER"
echo "POSTGRES_PASSWORD = $POSTGRES_PASSWORD"

# unprivileged user variables
echo "PGAPPNAME = $PGAPPNAME"
echo "PGDATABASE = $PGDATABASE"
echo "PGUSER = $PGUSER"
echo "PGPASSWORD = $PGPASSWORD"

if [ -z "$PGAPPNAME" -o -z "$PGDATABASE" -o -z "$PGUSER" -o -z "$PGPASSWORD" ]; then
	>&2 echo $0: "error: please set psql environment variables PGAPPNAME, PGDATABASE, PGUSER and PGPASSWORD"
	>&2 echo "for more information: https://www.postgresql.org/docs/current/libpq-envars.html"
	exit 1
fi

psql -a -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	CREATE USER ${PGUSER} with encrypted password '$PGPASSWORD';
	CREATE DATABASE ${PGDATABASE} LC_COLLATE 'fi_FI.utf8' LC_CTYPE 'fi_FI.utf8' TEMPLATE template0;
	GRANT ALL PRIVILEGES ON DATABASE ${PGDATABASE} TO ${PGUSER};

	\connect ${PGDATABASE};
	CREATE SCHEMA ${PGAPPNAME} AUTHORIZATION ${PGUSER};
	COMMENT ON SCHEMA ${PGAPPNAME} IS 'This is the main application SQL schema.';
	ALTER USER ${PGUSER} SET search_path = ${PGAPPNAME};
	REVOKE ALL PRIVILEGES ON SCHEMA public FROM ${PGUSER};
	REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC;
EOSQL

# don't let the postgres docker image run *.sql files automatically because we want to use our unprivileged user
SCHEMA_PATH="."
if [ -d "/docker-entrypoint-initdb.d/schemas" ]; then
	SCHEMA_PATH="/docker-entrypoint-initdb.d/schemas"
fi

# import SQL schema files into the database as the unprivileged user
for schema in $SQLFILES; do
	if [ -f "$SCHEMA_PATH/$schema" ]; then
		>&2 echo $0: "Processing SQL schema $SCHEMA_PATH/${schema}..."
		psql -a -v ON_ERROR_STOP=1 --username "$PGUSER" --dbname "$PGDATABASE" -f "$SCHEMA_PATH/${schema}"
	fi
done
