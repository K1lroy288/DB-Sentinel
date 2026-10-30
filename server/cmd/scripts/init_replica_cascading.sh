#!/bin/bash
set -exu

: "${PGDATA:=/var/lib/postgresql/data}"
export PGPASSWORD="$POSTGRES_PASSWORD"

# Ожидаем готовности промежуточной асинхронной реплики
while true; do
    IS_RECOVERY=$(psql -At -U "$REPLICATION_USER" -d "$POSTGRES_DB_NAME_MASTER" -h postgres-replica-async -c "SELECT pg_is_in_recovery();" 2>/dev/null || echo "error")
    if [ "$IS_RECOVERY" == "t" ]; then
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

gosu postgres pg_basebackup -h postgres-replica-async -p 5432 -U "$REPLICATION_USER" -D "$PGDATA" -Fp -X stream -C -S slot_replica_cascade -R -P

cat <<EOF >> "$PGDATA/postgresql.auto.conf"
primary_conninfo = 'host=postgres-replica-async port=5432 user=${REPLICATION_USER} password=${POSTGRES_PASSWORD} application_name=postgres-replica-cascading'
primary_slot_name = 'slot_replica_cascade'
EOF

exec gosu postgres postgres -D "$PGDATA"
