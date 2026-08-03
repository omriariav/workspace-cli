#!/bin/sh
# This adapter deliberately owns no configuration or provider logic. The Go
# runtime validates all commands, timezones, TTLs, and JSON output.
event="$1"
case "$event" in
  SessionStart|UserPromptSubmit) ;;
  *) exit 0 ;;
esac

if command -v gws >/dev/null 2>&1; then
  if output=$(env GWS_NO_UPDATE_CHECK=1 gws hooks run --event "$event" --adapter codex 2>/dev/null); then
    [ -n "$output" ] && printf '%s\n' "$output"
  fi
fi

exit 0
