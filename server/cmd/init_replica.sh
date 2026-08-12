#!/bin/bash 

set -exu

while ! pg_is_ready -U db_sentinel -d main_db -h postgres-master;
do
    sleep 1
done

if [[ -f "${PGDATA}/standby.signal" ||  -f "${PGDATA}/postgresql.conf" || -f "${PGDATA}/$PG_VERSION" ]]; then
    exec postgres -D /var/lib/postgresql/data
fi

rm -rf "${PGDATA:-"/tmp/some_catalog"}"

export PGPASSWORD="$POSTGRES_PASSWORD"

pg_basebackup -P -R -h postgres-master -p 5432

exec postgres -D /var/lib/postgresql/data