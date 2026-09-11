#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE ROLE ${REPLICATION_USER} WITH REPLICATION LOGIN PASSWORD '${REPLICATION_PASSWORD}';
    
    GRANT CONNECT ON DATABASE "${POSTGRES_DB}" TO ${REPLICATION_USER};
    GRANT USAGE ON SCHEMA public TO ${REPLICATION_USER};
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO ${REPLICATION_USER};
    
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO ${REPLICATION_USER};

    SELECT pg_create_physical_replication_slot('standby_slot');
EOSQL

echo "host replication ${REPLICATION_USER} all scram-sha-256" >> "$PGDATA/pg_hba.conf"

cat <<EOF >> "$PGDATA/postgresql.conf"
wal_level = replica
max_wal_senders = 5
max_replication_slots = 5
hot_standby = on
max_slot_wal_keep_size = 10GB
EOF

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT pg_reload_conf();"