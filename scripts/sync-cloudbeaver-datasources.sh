#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
DATA_SOURCES="$ROOT_DIR/cloudbeaver/data-sources.json"
PERMISSIONS="$ROOT_DIR/cloudbeaver/data-sources-permissions.json"

dbs=$(
  docker exec zxc-postgres \
    psql -U postgres -d postgres -At \
    -c "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;"
)

tmp_sources=$(mktemp)
tmp_permissions=$(mktemp)
trap 'rm -f "$tmp_sources" "$tmp_permissions"' EXIT

{
  printf '{\n'
  printf '  "folders": {},\n'
  printf '  "connections": {\n'

  first=1
  for db in $dbs; do
    [ "$first" -eq 1 ] || printf ',\n'
    first=0

    cat <<EOF
    "postgresql-$db": {
      "provider": "postgresql",
      "driver": "postgres-jdbc",
      "name": "$db",
      "save-password": true,
      "show-system-objects": true,
      "provider-properties": {
        "@dbeaver-show-non-default-db@": "true",
        "@dbeaver-show-template-db@": "false",
        "@dbeaver-show-unavailable-db@": "false",
        "show-database-statistics": "true",
        "@dbeaver-read-all-data-types-db@": "false",
        "read-keys-with-columns": "false",
        "@dbeaver-use-prepared-statements-db@": "false",
        "postgresql.dd.plain.string": "false",
        "postgresql.dd.tag.string": "false"
      },
      "configuration": {
        "host": "postgres",
        "port": "5432",
        "database": "$db",
        "user": "postgres",
        "password": "postgres",
        "configurationType": "MANUAL"
      }
    }
EOF
  done

  printf '\n  }\n'
  printf '}\n'
} >"$tmp_sources"

{
  printf '{\n'
  first=1
  for db in $dbs; do
    [ "$first" -eq 1 ] || printf ',\n'
    first=0
    printf '  "postgresql-%s": ["admin", "user"]' "$db"
  done
  printf '\n}\n'
} >"$tmp_permissions"

mv "$tmp_sources" "$DATA_SOURCES"
mv "$tmp_permissions" "$PERMISSIONS"

docker-compose -f "$ROOT_DIR/docker-compose.yml" restart cloudbeaver >/dev/null

echo "CloudBeaver datasources synced:"
for db in $dbs; do
  echo " - $db"
done
