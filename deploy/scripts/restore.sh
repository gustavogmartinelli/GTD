#!/usr/bin/env bash
# Runs ON THE VM as root. Restores a backup from Object Storage:
#   sudo /opt/gtd/restore.sh daily/gtd-2026-10-04.db.gz
#   sudo /opt/gtd/restore.sh            # lists available backups
# The current database is moved aside (never deleted) before the swap.
set -euo pipefail

# shellcheck disable=SC1091
source /etc/gtd/backup.env
DB=/var/lib/gtd/gtd.db

if [ $# -eq 0 ]; then
  echo "available backups:"
  rclone lsl "$RCLONE_DEST"
  echo; echo "usage: $0 <path, e.g. daily/gtd-YYYY-MM-DD.db.gz>"
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

rclone copyto "$RCLONE_DEST/$1" "$TMP/restore.db.gz"
gunzip "$TMP/restore.db.gz"
[ "$(sqlite3 "$TMP/restore.db" 'PRAGMA integrity_check;')" = "ok" ]
echo "downloaded $1: $(sqlite3 "$TMP/restore.db" 'SELECT COUNT(*) FROM items;') items"

cd /opt/gtd
docker compose stop gtd
if [ -f "$DB" ]; then
  KEEP="$DB.before-restore-$(date +%Y%m%d-%H%M%S)"
  mv "$DB" "$KEEP"
  echo "previous database kept at $KEEP"
fi
install -o 65532 -g 65532 -m 600 "$TMP/restore.db" "$DB"
docker compose start gtd

for _ in $(seq 1 30); do
  curl -fsS -m 2 http://127.0.0.1:8080/healthz >/dev/null 2>&1 && { echo "restored and healthy"; exit 0; }
  sleep 1
done
echo "!! app not healthy after restore; check: docker compose logs gtd"
exit 1
