#!/bin/bash 

set -exu

: "${PGDATA:=/var/lib/postgresql/data}"

export PGPASSWORD="$POSTGRES_PASSWORD"

while true; do
    IS_RECOVERY=$(psql -At -U "$REPLICATION_USER" -d "$PG_MASTER_DB_NAME" -h postgres-master -c "SELECT pg_is_in_recovery();" 2>/dev/null || echo "error")
    if [ "$IS_RECOVERY" == "f" ]; then
        break
    fi
    sleep 1
done

if [ -f "${PGDATA}/standby.signal" ]; then
    exec gosu postgres postgres -D "$PGDATA"
fi

mkdir -p "$PGDATA"

chown -R postgres:postgres "$PGDATA"
chmod 700 "$PGDATA"

find "$PGDATA" -mindepth 1 -maxdepth 1 ! -name ".lost+found" -exec rm -rf {} +

gosu postgres pg_basebackup -h postgres-master -p 5432 -U "$REPLICATION_USER" -D "$PGDATA" -Fp -X stream -C -S standby_slot_1 -R -P

exec gosu postgres postgres -D "$PGDATA"