#!/bin/sh
set -e

# Litestream continuously replicates the sqlite database to S3-compatible
# storage. Only the long-running server is wrapped: one-off commands such as
# `migrate` must not start a second replicating daemon against the same replica,
# and litestream neither locks nor detects that conflict.
if [ "$1" != "start" ] || [ -z "$LITESTREAM_REPLICA_URL" ]; then
    exec ./portrd "$@"
fi

case "$PORTR_DB_URL" in
    sqlite://* | sqlite3://*) ;;
    *) exec ./portrd "$@" ;;
esac

# Resolve the database path the same way internal/server/db/db.go does, and
# export it for ${PORTR_DB_PATH} expansion in /etc/litestream.yml. The Dockerfile
# sets the default so ad-hoc `litestream` commands resolve the same database.
PORTR_DB_PATH="${PORTR_DB_URL#*://}"
export PORTR_DB_PATH

# Litestream restores into this directory before portrd gets a chance to create it.
mkdir -p "$(dirname "$PORTR_DB_PATH")"

# portrd's arguments are flattened into the single string -exec expects, so they
# must not contain spaces.
exec litestream replicate -restore-if-db-not-exists -exec "./portrd $*"
