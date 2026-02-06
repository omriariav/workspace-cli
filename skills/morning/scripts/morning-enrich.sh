#!/bin/bash
# morning-enrich.sh — Deterministic enrichment of pre-filtered emails
#
# Usage:
#   morning-enrich.sh <scratchpad_dir> <config_file>
#
# Reads prefiltered.json (output of morning-prefilter.sh) and adds
# deterministic tags to each email. These tags reduce AI reasoning load
# by pre-computing signals that don't require semantic understanding.
#
# Input:
#   <scratchpad_dir>/prefiltered.json — emails after OOO/invite removal
#   <scratchpad_dir>/calendar.json — today's calendar events (optional)
#   <config_file> — inbox-skill.yaml with VIP senders list
#
# Output:
#   <scratchpad_dir>/enriched.json — emails with added tags object
#
# Tags added per email:
#   noise_signal: "promotions" | null (CATEGORY_PROMOTIONS label)
#   vip_sender: true | false (sender in VIP list from config)
#   starred: true | false (STARRED label present)
#   is_thread: true | false (message_count > 1)
#   calendar_match: "<event title>" | null (fuzzy subject match)
#
# Requires: jq

set -euo pipefail

SCRATCHPAD_DIR="${1:-}"
CONFIG_FILE="${2:-$HOME/.config/gws/inbox-skill.yaml}"

if [ -z "$SCRATCHPAD_DIR" ]; then
  echo '{"error":"Usage: morning-enrich.sh <scratchpad_dir> [config_file]"}'
  exit 1
fi

if ! command -v jq &>/dev/null; then
  echo '{"error":"jq is required but not found"}'
  exit 1
fi

PREFILTERED="$SCRATCHPAD_DIR/prefiltered.json"
CALENDAR="$SCRATCHPAD_DIR/calendar.json"
ENRICHED="$SCRATCHPAD_DIR/enriched.json"

if [ ! -f "$PREFILTERED" ]; then
  echo '{"error":"prefiltered.json not found"}'
  exit 1
fi

# --- Extract VIP senders from config ---

VIP_SENDERS='[]'
if [ -f "$CONFIG_FILE" ]; then
  # Parse YAML VIP senders list (simple grep-based extraction)
  VIP_SENDERS=$(grep -A 100 'vip_senders:' "$CONFIG_FILE" 2>/dev/null | \
    grep '^\s*-\s' | \
    sed 's/^\s*-\s*//' | \
    sed 's/#.*//' | \
    sed 's/\s*$//' | \
    jq -R -s 'split("\n") | map(select(length > 0) | ascii_downcase)' 2>/dev/null || echo '[]')
fi

# --- Extract calendar event titles ---

CAL_TITLES='[]'
if [ -f "$CALENDAR" ]; then
  CAL_TITLES=$(jq '[.events[]?.summary // empty] | map(ascii_downcase)' "$CALENDAR" 2>/dev/null || echo '[]')
fi

# --- Enrich each email ---

TOTAL=$(jq 'length' "$PREFILTERED")
ENRICHED_ARRAY='[]'

NOISE_COUNT=0
VIP_COUNT=0
STARRED_COUNT=0
THREAD_COUNT=0
CAL_MATCH_COUNT=0

for i in $(seq 0 $((TOTAL - 1))); do
  ENTRY=$(jq ".[$i]" "$PREFILTERED")

  # Extract fields
  SENDER=$(echo "$ENTRY" | jq -r '(.sender // .from // "") | ascii_downcase')
  SUBJECT=$(echo "$ENTRY" | jq -r '(.subject // "") | ascii_downcase')
  MSG_COUNT=$(echo "$ENTRY" | jq -r '.message_count // 1')
  LABELS=$(echo "$ENTRY" | jq -r '.labels // [] | map(ascii_downcase) | join(" ")')
  SNIPPET=$(echo "$ENTRY" | jq -r '.snippet // ""')

  # --- Tag: noise_signal ---
  NOISE_SIGNAL="null"
  if echo "$LABELS" | grep -q "category_promotions"; then
    NOISE_SIGNAL='"promotions"'
    NOISE_COUNT=$((NOISE_COUNT + 1))
  fi

  # --- Tag: vip_sender ---
  VIP="false"
  SENDER_EMAIL=$(echo "$SENDER" | grep -oE '[a-z0-9._%+-]+@[a-z0-9.-]+' | head -1)
  if [ -n "$SENDER_EMAIL" ]; then
    if echo "$VIP_SENDERS" | jq -e --arg email "$SENDER_EMAIL" 'index($email) != null' >/dev/null 2>&1; then
      VIP="true"
      VIP_COUNT=$((VIP_COUNT + 1))
    fi
  fi

  # --- Tag: starred ---
  STARRED="false"
  if echo "$LABELS" | grep -q "starred"; then
    STARRED="true"
    STARRED_COUNT=$((STARRED_COUNT + 1))
  fi

  # --- Tag: is_thread ---
  IS_THREAD="false"
  if [ "$MSG_COUNT" -gt 1 ] 2>/dev/null; then
    IS_THREAD="true"
    THREAD_COUNT=$((THREAD_COUNT + 1))
  fi

  # --- Tag: calendar_match ---
  CAL_MATCH="null"
  if [ "$CAL_TITLES" != "[]" ] && [ -n "$SUBJECT" ]; then
    # Check each calendar title for word overlap with subject
    MATCH=$(echo "$CAL_TITLES" | jq -r --arg subj "$SUBJECT" '
      . as $titles |
      ($subj | split(" ") | map(select(length > 3))) as $words |
      [range(length)] |
      map(
        . as $idx |
        $titles[$idx] as $title |
        ($title | split(" ") | map(select(length > 3))) as $title_words |
        ($words | map(. as $w | $title_words | map(select(contains($w))) | length > 0) | map(select(.)) | length) as $overlap |
        if $overlap >= 2 then $title else empty end
      ) | first // empty
    ' 2>/dev/null || true)
    if [ -n "$MATCH" ]; then
      CAL_MATCH=$(echo "$MATCH" | jq -R '.')
      CAL_MATCH_COUNT=$((CAL_MATCH_COUNT + 1))
    fi
  fi

  # --- Build enriched entry ---
  ENRICHED_ENTRY=$(echo "$ENTRY" | jq \
    --argjson noise "$NOISE_SIGNAL" \
    --argjson vip "$VIP" \
    --argjson starred "$STARRED" \
    --argjson is_thread "$IS_THREAD" \
    --argjson cal_match "$CAL_MATCH" \
    '. + {"tags": {"noise_signal": $noise, "vip_sender": $vip, "starred": $starred, "is_thread": $is_thread, "calendar_match": $cal_match}}')

  ENRICHED_ARRAY=$(echo "$ENRICHED_ARRAY" | jq --argjson entry "$ENRICHED_ENTRY" '. + [$entry]')
done

# --- Write output ---

echo "$ENRICHED_ARRAY" > "$ENRICHED"

# --- Summary ---

cat <<EOF
{"total":$TOTAL,"noise_tagged":$NOISE_COUNT,"vip_tagged":$VIP_COUNT,"starred_tagged":$STARRED_COUNT,"threads_tagged":$THREAD_COUNT,"calendar_matched":$CAL_MATCH_COUNT}
EOF
