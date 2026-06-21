#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"
if [ -n "${PRINT_IT_PKG_ARCH:-}" ]; then
  GOARCH="$PRINT_IT_PKG_ARCH"
else
  ARCH="$(uname -m)"
  case "$ARCH" in
    arm64) GOARCH=arm64 ;;
    x86_64) GOARCH=amd64 ;;
    *)
      echo "Arquitetura macOS nao suportada: $ARCH" >&2
      exit 1
      ;;
  esac
fi

BINARY_SRC="$ROOT/dist/print.it-darwin-$GOARCH"
if [ ! -f "$BINARY_SRC" ]; then
  echo ">> Compilando print.it-darwin-$GOARCH..."
  eval "$(packaging/read-build-config.sh export)"
  GOOS=darwin GOARCH="$GOARCH" go build -ldflags "-s -w ${PRINT_IT_LDFLAGS_BUILD}" -o "$BINARY_SRC" .
fi

PKGROOT="$ROOT/dist/macos-pkgroot"
SCRIPTS="$ROOT/dist/macos-scripts"
ICONSET="$ROOT/dist/appicon.iconset"
APP="$PKGROOT/Applications/print.it.app"
OUT="$ROOT/dist/print.it-${VERSION}-macos-${GOARCH}.pkg"

rm -rf "$PKGROOT" "$SCRIPTS" "$ICONSET"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$PKGROOT/usr/local/share/print.it" "$ICONSET" "$SCRIPTS"

cp "$BINARY_SRC" "$APP/Contents/MacOS/print-it-agent"
chmod 755 "$APP/Contents/MacOS/print-it-agent"

cat > "$APP/Contents/MacOS/print.it" <<'EOF'
#!/bin/bash
APP_DIR="$(cd "$(dirname "$0")" && pwd)"
AGENT="$APP_DIR/print-it-agent"
PLIST="$HOME/Library/LaunchAgents/com.printit.agent.plist"
UID_NUM="$(id -u)"

if [ ! -x "$AGENT" ]; then
  osascript -e 'display alert "print.it" message "Agente nao encontrado."' 2>/dev/null || true
  exit 1
fi

if [ -f "$PLIST" ]; then
  launchctl kickstart -k "gui/$UID_NUM/com.printit.agent" 2>/dev/null || \
    launchctl bootstrap "gui/$UID_NUM" "$PLIST" 2>/dev/null || true
else
  "$AGENT" &
fi

for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  if curl -sf --max-time 1 http://127.0.0.1:9280/printit/health >/dev/null 2>&1; then
    open "http://127.0.0.1:9280/printit/"
    exit 0
  fi
  sleep 0.4
done

osascript -e 'display alert "print.it" message "O agente nao respondeu. Verifique ~/Library/Logs/print.it/"' 2>/dev/null || true
exit 1
EOF
chmod 755 "$APP/Contents/MacOS/print.it"

if [ ! -f packaging/appicon.png ]; then
  echo "packaging/appicon.png nao encontrado" >&2
  exit 1
fi

for size in 16 32 128 256 512; do
  sips -z "$size" "$size" packaging/appicon.png --out "$ICONSET/icon_${size}x${size}.png" >/dev/null
  if [ "$size" -le 256 ]; then
    double=$((size * 2))
    sips -z "$double" "$double" packaging/appicon.png --out "$ICONSET/icon_${size}x${size}@2x.png" >/dev/null
  fi
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/appicon.icns"

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>print.it</string>
  <key>CFBundleIconFile</key>
  <string>appicon</string>
  <key>CFBundleIdentifier</key>
  <string>com.printit.agent</string>
  <key>CFBundleName</key>
  <string>print.it</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
</dict>
</plist>
EOF

cp packaging/macos/com.printit.agent.plist "$PKGROOT/usr/local/share/print.it/"
cp packaging/macos/uninstall.sh "$PKGROOT/usr/local/share/print.it/"
chmod 755 "$PKGROOT/usr/local/share/print.it/uninstall.sh"
cp packaging/macos/postinstall "$SCRIPTS/"
cp packaging/macos/preinstall "$SCRIPTS/"
chmod 755 "$SCRIPTS/postinstall" "$SCRIPTS/preinstall"

COMPONENT="$ROOT/dist/print.it-component.pkg"
pkgbuild \
  --root "$PKGROOT" \
  --scripts "$SCRIPTS" \
  --identifier "com.printit.agent" \
  --version "$VERSION" \
  --install-location "/" \
  "$COMPONENT"

productbuild \
  --package "$COMPONENT" \
  "$OUT"

echo ""
echo "Pacote gerado: $OUT"
