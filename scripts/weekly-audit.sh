#!/bin/sh
# Weekly dependency audit for machines where GitHub Actions is not available.
# Run by a systemd user timer (see AGENTS.md). Appends the result to
# ~/.local/state/hydrogo-audit.log and raises a desktop notification on findings.
set -u
repo="$(cd "$(dirname "$0")/.." && pwd)"
log="${XDG_STATE_HOME:-$HOME/.local/state}/hydrogo-audit.log"
mkdir -p "$(dirname "$log")"
cd "$repo" || exit 1
{ echo "=== $(date -Is)"; go run ./tools/check -short -audit; } >> "$log" 2>&1
status=$?
if [ "$status" -ne 0 ]; then
  command -v notify-send >/dev/null 2>&1 && notify-send "HydroGo audit" "Dependency audit found something. See $log"
fi
exit "$status"
