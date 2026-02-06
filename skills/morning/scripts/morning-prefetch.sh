#!/bin/bash
# morning-prefetch.sh — Prefetch all data sources for /morning skill
#
# Usage:
#   morning-prefetch.sh <output_dir> [--refresh-okr]
#
# Reads config from ~/.config/gws/inbox-skill.yaml (no yq dependency).
# Fetches Gmail inbox, Calendar, Tasks, and OKR sheets in parallel.
# OKRs are cached for 24 hours at ~/.cache/gws/morning/okr_*.json.
# Use --refresh-okr to force a fresh OKR fetch.
# Output: JSON summary on stdout with status of each data source.

set -euo pipefail

OUTPUT_DIR="${1:-}"
REFRESH_OKR=false

for arg in "$@"; do
  case "$arg" in
    --refresh-okr) REFRESH_OKR=true ;;
  esac
done

if [ -z "$OUTPUT_DIR" ]; then
  echo '{"error":"Usage: morning-prefetch.sh <output_dir> [--refresh-okr]"}'
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

START_TIME=$(date +%s)

# --- Config ---

CONFIG_FILE="$HOME/.config/gws/inbox-skill.yaml"

cfg_get() {
  local key="$1"
  local default="${2:-}"
  if [ -f "$CONFIG_FILE" ]; then
    local val
    val=$(grep "^${key}:" "$CONFIG_FILE" 2>/dev/null | head -1 | sed "s/^${key}:[[:space:]]*//" | sed 's/^"//' | sed 's/"$//' | sed "s/^'//" | sed "s/'$//")
    if [ -n "$val" ]; then
      echo "$val"
      return
    fi
  fi
  echo "$default"
}

cfg_get_list() {
  local key="$1"
  if [ -f "$CONFIG_FILE" ]; then
    awk "/^${key}:/{found=1; next} found && /^  - /{gsub(/^  - /,\"\"); gsub(/\"/, \"\"); print; next} found && /^[^ ]/{exit}" "$CONFIG_FILE"
  fi
}

# Read config values
MAX_EMAILS=$(cfg_get "max_unread" "")
if [ -z "$MAX_EMAILS" ]; then
  MAX_EMAILS=$(cfg_get "max_emails" "50")
fi

OKR_SHEET_ID=$(cfg_get "okr_sheet_id" "")

# Read okr_sheets list
OKR_SHEETS=()
while IFS= read -r line; do
  [ -n "$line" ] && OKR_SHEETS+=("$line")
done < <(cfg_get_list "okr_sheets")

# --- jq check ---

HAS_JQ=false
if command -v jq &>/dev/null; then
  HAS_JQ=true
fi

json_count() {
  local file="$1"
  if $HAS_JQ && [ -f "$file" ]; then
    jq 'if type == "array" then length elif .count? then .count elif .threads? then (.threads | length) elif .events? then (.events | length) elif .tasklists? then (.tasklists | length) elif .values? then (.values | length) else 1 end' "$file" 2>/dev/null || echo "null"
  else
    echo "null"
  fi
}

# --- OKR Cache ---

OKR_CACHE_DIR="$HOME/.cache/gws/morning"
OKR_CACHE_TTL=86400  # 24 hours in seconds

okr_cache_fresh() {
  local idx="$1"
  local cache_file="$OKR_CACHE_DIR/okr_${idx}.json"
  if [ ! -f "$cache_file" ]; then
    return 1
  fi
  local file_age
  file_age=$(( $(date +%s) - $(stat -f %m "$cache_file" 2>/dev/null || stat -c %Y "$cache_file" 2>/dev/null || echo 0) ))
  [ "$file_age" -lt "$OKR_CACHE_TTL" ]
}

# --- Phase 1: Parallel fetches ---

# Inbox
(
  gws gmail list --max "$MAX_EMAILS" --query "is:unread in:inbox" --include-labels > "$OUTPUT_DIR/inbox.json" 2> "$OUTPUT_DIR/inbox.err"
) &

# Calendar
(
  gws calendar events --days 2 > "$OUTPUT_DIR/calendar.json" 2> "$OUTPUT_DIR/calendar.err"
) &

# Task lists
(
  gws tasks lists > "$OUTPUT_DIR/tasks.json" 2> "$OUTPUT_DIR/tasks.err"
) &

# OKR sheets (cached)
okr_idx=0
okr_cached=0
mkdir -p "$OKR_CACHE_DIR"
for sheet_name in "${OKR_SHEETS[@]}"; do
  if [ -n "$OKR_SHEET_ID" ]; then
    if ! $REFRESH_OKR && okr_cache_fresh "$okr_idx"; then
      # Use cached copy
      cp "$OKR_CACHE_DIR/okr_${okr_idx}.json" "$OUTPUT_DIR/okr_${okr_idx}.json"
      okr_cached=$((okr_cached + 1))
    else
      # Fetch fresh and update cache
      (
        gws sheets read "$OKR_SHEET_ID" "${sheet_name}!A1:Q100" > "$OUTPUT_DIR/okr_${okr_idx}.json" 2> "$OUTPUT_DIR/okr_${okr_idx}.err"
        if [ -s "$OUTPUT_DIR/okr_${okr_idx}.json" ]; then
          cp "$OUTPUT_DIR/okr_${okr_idx}.json" "$OKR_CACHE_DIR/okr_${okr_idx}.json"
        fi
      ) &
    fi
    okr_idx=$((okr_idx + 1))
  fi
done

wait

# --- Phase 2: Fetch individual task lists (depends on tasks.json) ---

if [ -f "$OUTPUT_DIR/tasks.json" ] && $HAS_JQ; then
  task_list_ids=$(jq -r '(.tasklists // .[])[] | .id // empty' "$OUTPUT_DIR/tasks.json" 2>/dev/null || true)
  for list_id in $task_list_ids; do
    (
      gws tasks list "$list_id" > "$OUTPUT_DIR/tasks_${list_id}.json" 2> "$OUTPUT_DIR/tasks_${list_id}.err"
    ) &
  done
  wait
fi

# --- Phase 3: Extract overdue tasks (deterministic, no AI needed) ---

overdue_count=0
if $HAS_JQ; then
  TODAY=$(date +%Y-%m-%dT00:00:00.000Z)
  # Scan all task list files for tasks with due date before today
  jq -n --arg today "$TODAY" '
    [inputs | (.tasks // [])[] | select(.due != null and .due < $today and .status == "needsAction")]
  ' "$OUTPUT_DIR"/tasks_*.json 2>/dev/null > "$OUTPUT_DIR/overdue.json" || echo "[]" > "$OUTPUT_DIR/overdue.json"
  overdue_count=$(jq 'length' "$OUTPUT_DIR/overdue.json" 2>/dev/null || echo 0)
fi

# --- Summary ---

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

# Compute statuses
inbox_status="error"
[ -f "$OUTPUT_DIR/inbox.json" ] && [ -s "$OUTPUT_DIR/inbox.json" ] && inbox_status="success"
inbox_count=$(json_count "$OUTPUT_DIR/inbox.json")

calendar_status="error"
[ -f "$OUTPUT_DIR/calendar.json" ] && [ -s "$OUTPUT_DIR/calendar.json" ] && calendar_status="success"
calendar_count=$(json_count "$OUTPUT_DIR/calendar.json")

tasks_status="error"
[ -f "$OUTPUT_DIR/tasks.json" ] && [ -s "$OUTPUT_DIR/tasks.json" ] && tasks_status="success"
tasks_lists=$(json_count "$OUTPUT_DIR/tasks.json")

okr_status="skipped"
okr_source="none"
if [ "$okr_idx" -gt 0 ] 2>/dev/null; then
  okr_status="error"
  if [ -f "$OUTPUT_DIR/okr_0.json" ] && [ -s "$OUTPUT_DIR/okr_0.json" ]; then
    okr_status="success"
    if [ "$okr_cached" -eq "$okr_idx" ]; then
      okr_source="cache"
    elif [ "$okr_cached" -gt 0 ]; then
      okr_source="partial_cache"
    else
      okr_source="fresh"
    fi
  fi
fi

cat <<EOF
{"output_dir":"$OUTPUT_DIR","fetch_time_seconds":$ELAPSED,"inbox":{"status":"$inbox_status","count":$inbox_count},"calendar":{"status":"$calendar_status","count":$calendar_count},"tasks":{"status":"$tasks_status","lists":$tasks_lists,"overdue":$overdue_count},"okr":{"status":"$okr_status","sheets":${okr_idx:-0},"source":"$okr_source"}}
EOF
