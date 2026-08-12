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
    exec postgres -D ${PGDATA:-"/var/lib/postgresql/data"}
fi

find "$PGDATA" -mindepth 1 -maxdepth 1 ! -name ".lost+found" -exec rm -rf {} +

pg_basebackup -P -R -h postgres-master -p 5432

exec postgres -D ${PGDATA:-"/var/lib/postgresql/data"}