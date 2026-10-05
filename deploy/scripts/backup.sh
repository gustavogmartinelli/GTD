#!/usr/bin/env bash
# Runs ON THE VM as root from /etc/cron.d/gtd-backup (nightly).
# Takes a consistent SQLite snapshot, verifies it, uploads it to OCI
# Object Storage via rclone, and pings healthchecks.io. A missing ping
# makes healthchecks.io email you.
#
# Layout in the bucket:
#   daily/gtd-YYYY-MM-DD.db.gz    deleted after 30 days by a lifecycle rule
#   monthly/gtd-YYYY-MM.db.gz     kept forever (taken on the 1st)
#
# Config: /etc/gtd/backup.env (see docs/deploy/oracle-setup.md)
#   RCLONE_DEST=oci:gtd-backups
#   HC_PING_URL=https://hc-ping.com/<uuid>
set -euo pipefail

# shellcheck disable=SC1091
source /etc/gtd/backup.env
DB=/var/lib/gtd/gtd.db

ping_hc() { [ -n "${HC_PING_URL:-}" ] && curl -fsS -m 10 --retry 3 "$HC_PING_URL$1" >/dev/null || true; }
trap 'ping_hc /fail' ERR
ping_hc /start

# Opening a missing path would make sqlite3 create a root-owned empty
# file that the app (uid 65532) then cannot write to.
if [ ! -f "$DB" ]; then
  echo "no database at $DB yet; nothing to back up"
  ping_hc ""
  exit 0
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

DAY="$(date +%F)"
SNAP="$TMP/gtd-$DAY.db"

# .backup uses SQLite's online backup API: safe while the app is writing.
sqlite3 "$DB" ".backup '$SNAP'"
[ "$(sqlite3 "$SNAP" 'PRAGMA integrity_check;')" = "ok" ]
gzip -9 "$SNAP"

rclone copyto "$SNAP.gz" "$RCLONE_DEST/daily/gtd-$DAY.db.gz"
if [ "$(date +%d)" = "01" ]; then
  rclone copyto "$SNAP.gz" "$RCLONE_DEST/monthly/gtd-$(date +%Y-%m).db.gz"
fi

echo "backup ok: $DAY ($(stat -c %s "$SNAP.gz") bytes)"
ping_hc ""
