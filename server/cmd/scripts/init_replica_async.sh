#!/bin/bash
set -exu

: "${PGDATA:=/var/lib/postgresql/data}"
export PGPASSWORD="$POSTGRES_PASSWORD"

while true; do
    IS_RECOVERY=$(psql -At -U "$REPLICATION_USER" -d "$POSTGRES_DB_NAME_MASTER" -h postgres-master -c "SELECT pg_is_in_recovery();" 2>/dev/null || echo "error")
    if [ "$IS_RECOVERY" == "f" ]; then
        break
    fi
    sleep 1
done

if [ -f "${PGDATA}/standby.signal" ]; then
    sed -i '/max_wal_senders/d' "$PGDATA/postgresql.auto.conf" 2>/dev/null || true
    sed -i '/max_replication_slots/d' "$PGDATA/postgresql.auto.conf" 2>/dev/null || true
    echo "max_wal_senders = 10" >> "$PGDATA/postgresql.auto.conf"
    echo "max_replication_slots = 10" >> "$PGDATA/postgresql.auto.conf"
    exec gosu postgres postgres -D "$PGDATA" -c max_wal_senders=10 -c max_replication_slots=10
fi

mkdir -p "$PGDATA"
chown -R postgres:postgres "$PGDATA"
chmod 700 "$PGDATA"

find "$PGDATA" -mindepth 1 -maxdepth 1 ! -name ".lost+found" -exec rm -rf {} +

gosu postgres pg_basebackup -h postgres-master -p 5432 -U "$REPLICATION_USER" -D "$PGDATA" -Fp -X stream -C -S slot_replica_async -R -P

cat <<EOF >> "$PGDATA/postgresql.auto.conf"
primary_conninfo = 'host=postgres-master port=5432 user=${REPLICATION_USER} password=${POSTGRES_PASSWORD} application_name=postgres-replica-async'
primary_slot_name = 'slot_replica_async'
hot_standby = on
max_wal_senders = 10
max_replication_slots = 10
EOF

echo "host replication ${REPLICATION_USER} all scram-sha-256" >> "$PGDATA/pg_hba.conf"

exec gosu postgres postgres -D "$PGDATA" -c max_wal_senders=10 -c max_replication_slots=10
