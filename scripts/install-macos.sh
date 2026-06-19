#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

"$ROOT/scripts/build.sh"

PLIST="$HOME/Library/LaunchAgents/com.printit.agent.plist"
BINARY="$ROOT/print.it"
LOG_DIR="$HOME/Library/Logs/print.it"

mkdir -p "$LOG_DIR"

cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.printit.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>${BINARY}</string>
  </array>
  <key>WorkingDirectory</key>
  <string>${ROOT}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>${LOG_DIR}/stdout.log</string>
  <key>StandardErrorPath</key>
  <string>${LOG_DIR}/stderr.log</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/com.printit.agent" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$PLIST"
launchctl enable "gui/$(id -u)/com.printit.agent"
launchctl kickstart -k "gui/$(id -u)/com.printit.agent"

echo ""
echo "print.it instalado para iniciar automaticamente no login."
echo "Logs: $LOG_DIR"
echo "API:  http://127.0.0.1:9280/health"
