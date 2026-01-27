#!/bin/bash
# bulk-gmail.sh — Bulk Gmail operations for /morning skill
#
# Usage:
#   bulk-gmail.sh archive <id1> <id2> ...    # archive + mark read
#   bulk-gmail.sh trash <id1> <id2> ...      # delete + mark read
#   bulk-gmail.sh mark-read <id1> <id2> ...  # mark read only
#
# All actions except mark-read also mark emails as read.
# Output: JSON summary { action, total, success, failed }

set -euo pipefail

ACTION="${1:-}"
shift 2>/dev/null || true

if [ -z "$ACTION" ] || [ $# -eq 0 ]; then
  echo '{"error":"Usage: bulk-gmail.sh <archive|trash|mark-read> <id1> [id2] ..."}'
  exit 1
fi

case "$ACTION" in
  archive|trash|mark-read) ;;
  *)
    echo '{"error":"Unknown action: '"$ACTION"'","valid_actions":["archive","trash","mark-read"]}'
    exit 1
    ;;
esac

success=0
failed=0

for id in "$@"; do
  ok=true

  case "$ACTION" in
    archive)
      gws gmail archive "$id" >/dev/null 2>&1 || ok=false
      ;;
    trash)
      gws gmail trash "$id" >/dev/null 2>&1 || ok=false
      ;;
    mark-read)
      ;;
  esac

  # Mark as read (all actions do this)
  if $ok; then
    gws gmail label "$id" --remove UNREAD >/dev/null 2>&1 || ok=false
  fi

  if $ok; then
    ((success++))
  else
    ((failed++))
  fi
done

echo "{\"action\":\"$ACTION\",\"total\":$((success + failed)),\"success\":$success,\"failed\":$failed}"
