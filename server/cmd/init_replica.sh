#!/bin/bash 

set -exu

: "${PGDATA:=/var/lib/postgresql/data}"

while ! pg_isready -U db_sentinel -d main_db -h postgres-master;
do
    sleep 1
done

export PGPASSWORD="$POSTGRES_PASSWORD"

while [ "$(psql -At -U db_sentinel -d main_db -h postgres-master -c "SELECT pg_is_in_recovery();")" == "t" ];
do
    sleep 1
done

if [ -f "${PGDATA}/standby.signal" ]; then
    exec postgres -D "$PGDATA"
fi

mkdir -p "$PGDATA"

find "$PGDATA" -mindepth 1 -maxdepth 1 ! -name ".lost+found" -exec rm -rf {} +

pg_basebackup -h postgres-master -p 5432 -U replicator -D "$PGDATA" -Fp -Xs -R -P

exec postgres -D "$PGDATA"