#!/bin/sh
# Creates a Postgres role and database <name> in the shared db stack, with a
# generated password written only to stacks/<name>/.env (gitignored). The
# password never goes on a command line or to stdout.
#   stacks/db/new-database.sh <name>
set -eu

name=${1:?usage: $0 <name>}
case $name in *[!a-z0-9_]* | '') echo "name must be [a-z0-9_]" >&2; exit 1 ;; esac

env_file="$(dirname "$0")/../$name/.env"
if [ -f "$env_file" ] && grep -q '^DB_PASSWORD=' "$env_file"; then
  echo "$env_file already has DB_PASSWORD; not overwriting" >&2
  exit 1
fi

# Hex only: safe in SQL quotes and URLs without escaping.
password=$(openssl rand -hex 24)

# Fails (ON_ERROR_STOP) if the role or database already exists, before any
# file is written.
printf "CREATE ROLE %s LOGIN PASSWORD '%s';\nCREATE DATABASE %s OWNER %s;\n" \
  "$name" "$password" "$name" "$name" |
  docker --context blackberry exec -i db-postgres-1 psql -q -U postgres -v ON_ERROR_STOP=1

mkdir -p "$(dirname "$env_file")"
umask 077
cat >>"$env_file" <<EOF
DB_PASSWORD=$password
# From containers on the db network
DATABASE_URL=postgresql://$name:$password@postgres:5432/$name
# From the dev machine, through the ssh tunnel (stacks/README.md#shared-database)
LOCAL_DATABASE_URL=postgresql://$name:$password@127.0.0.1:5432/$name
EOF

echo "created role and database $name; password in $env_file"
