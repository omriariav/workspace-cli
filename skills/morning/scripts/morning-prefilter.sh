#!/bin/bash
# morning-prefilter.sh — Pre-filter inbox emails before AI classification
#
# Usage:
#   morning-prefilter.sh <prefetch_output_dir>
#
# Deterministically archives OOO replies and calendar invite emails,
# then writes remaining emails to prefiltered.json for AI classification.
# Requires jq.

set -euo pipefail

OUTPUT_DIR="${1:-}"

if [ -z "$OUTPUT_DIR" ]; then
  echo '{"error":"Usage: morning-prefilter.sh <prefetch_output_dir>"}'
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo '{"error":"jq is required but not found"}'
  exit 1
fi

INBOX_FILE="$OUTPUT_DIR/inbox.json"

if [ ! -f "$INBOX_FILE" ]; then
  echo '{"error":"inbox.json not found in output directory"}'
  exit 1
fi

# --- Pattern Matching ---

# Extract threads array from inbox.json (handles both {threads:[...]} and [...] formats)
THREADS=$(jq 'if type == "array" then . elif .threads then .threads else [] end' "$INBOX_FILE")
TOTAL=$(echo "$THREADS" | jq 'length')

AUTO_HANDLED='[]'
REMAINING='[]'
OOO_COUNT=0
INVITE_COUNT=0

for i in $(seq 0 $((TOTAL - 1))); do
  ENTRY=$(echo "$THREADS" | jq ".[$i]")
  SUBJECT=$(echo "$ENTRY" | jq -r '.subject // ""')
  SENDER=$(echo "$ENTRY" | jq -r '.sender // .from // ""')
  THREAD_ID=$(echo "$ENTRY" | jq -r '.thread_id // .id // ""')

  REASON=""

  # Pattern 1: OOO replies (case-insensitive)
  # Matches: "Out of Office", "OOO Re:", "out of the office", "Automatic reply:"
  SUBJECT_LOWER=$(echo "$SUBJECT" | tr '[:upper:]' '[:lower:]')
  if echo "$SUBJECT_LOWER" | grep -qE '(out of office|ooo re:|out of the office|automatic reply:)'; then
    REASON="ooo_reply"
    OOO_COUNT=$((OOO_COUNT + 1))
  fi

  # Pattern 2: Calendar invite emails (redundant — we have events from calendar API)
  if [ -z "$REASON" ]; then
    if echo "$SUBJECT" | grep -qE '^(Invitation:|Updated Invitation:|Canceled:)'; then
      if echo "$SENDER" | grep -qi 'calendar-notification@google.com'; then
        REASON="calendar_invite"
        INVITE_COUNT=$((INVITE_COUNT + 1))
      fi
    fi
  fi

  if [ -n "$REASON" ]; then
    # Archive the thread
    gws gmail archive-thread "$THREAD_ID" --quiet 2>/dev/null || true
    AUTO_HANDLED=$(echo "$AUTO_HANDLED" | jq --argjson entry "$ENTRY" --arg reason "$REASON" '. + [$entry + {"prefilter_reason": $reason}]')
  else
    REMAINING=$(echo "$REMAINING" | jq --argjson entry "$ENTRY" '. + [$entry]')
  fi
done

# --- Write output files ---

echo "$REMAINING" > "$OUTPUT_DIR/prefiltered.json"
echo "$AUTO_HANDLED" > "$OUTPUT_DIR/auto_handled.json"

# --- Summary ---

ARCHIVED=$((OOO_COUNT + INVITE_COUNT))
REMAINING_COUNT=$(echo "$REMAINING" | jq 'length')

cat <<EOF
{"total":$TOTAL,"auto_archived":$ARCHIVED,"remaining":$REMAINING_COUNT,"ooo":$OOO_COUNT,"invites":$INVITE_COUNT}
EOF
