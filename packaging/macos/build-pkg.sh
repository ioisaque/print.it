#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"
ARCH="$(uname -m)"
case "$ARCH" in
  arm64) GOARCH=arm64 ;;
  x86_64) GOARCH=amd64 ;;
  *)
    echo "Arquitetura macOS nao suportada: $ARCH" >&2
    exit 1
    ;;
esac

BINARY_SRC="$ROOT/dist/print.it-darwin-$GOARCH"
if [ ! -f "$BINARY_SRC" ]; then
  echo ">> Compilando print.it-darwin-$GOARCH..."
  GOOS=darwin GOARCH="$GOARCH" go build -ldflags "-s -w" -o "$BINARY_SRC" .
fi

PKGROOT="$ROOT/dist/macos-pkgroot"
SCRIPTS="$ROOT/dist/macos-scripts"
OUT="$ROOT/dist/print.it-${VERSION}-macos-${GOARCH}.pkg"

rm -rf "$PKGROOT" "$SCRIPTS"
mkdir -p "$PKGROOT/usr/local/bin" "$PKGROOT/usr/local/share/print.it" "$SCRIPTS"

cp "$BINARY_SRC" "$PKGROOT/usr/local/bin/print.it"
chmod 755 "$PKGROOT/usr/local/bin/print.it"
cp packaging/macos/com.printit.agent.plist "$PKGROOT/usr/local/share/print.it/"
cp packaging/macos/uninstall.sh "$PKGROOT/usr/local/share/print.it/"
chmod 755 "$PKGROOT/usr/local/share/print.it/uninstall.sh"
cp packaging/macos/postinstall "$SCRIPTS/"
chmod 755 "$SCRIPTS/postinstall"

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
