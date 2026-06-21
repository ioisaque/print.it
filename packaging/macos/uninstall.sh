#!/usr/bin/env bash
set -euo pipefail

BINARY="/usr/local/bin/print.it"
SHARE="/usr/local/share/print.it"
PLIST="$HOME/Library/LaunchAgents/com.printit.agent.plist"
USER_ID="$(id -u)"

if [ -f "$PLIST" ]; then
  launchctl bootout "gui/$USER_ID/com.printit.agent" 2>/dev/null || true
  rm -f "$PLIST"
fi

rm -f "$BINARY"
rm -rf "$SHARE"

echo "print.it desinstalado."
