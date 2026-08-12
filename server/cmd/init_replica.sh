#!/bin/bash 

set -exu

while ! pg_is_ready -U db_sentinel -d main_db -h postgres-master;
do
    sleep 1
done

export PGPASSWORD="$POSTGRES_PASSWORD"

while [ "$(psql -At -U db_sentinel -h postgres-master -c "SELECT pg_is_in_recovery();")" == "t" ];
do
    sleep 1
done

if [ -f "${PGDATA}/standby.signal" ]; then
    exec postgres -D $PGDATA
fi

rm -rf "${PGDATA:-"/tmp/some_catalog/"}"/*

pg_basebackup -P -R -h postgres-master -p 5432

exec postgres -D $PGDATA