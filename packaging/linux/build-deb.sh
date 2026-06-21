#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"
BINARY_SRC="$ROOT/dist/print.it-linux-amd64"
STAGE="$ROOT/dist/debian"
OUT="$ROOT/dist/print.it-${VERSION}-linux-amd64.deb"

if [ ! -f "$BINARY_SRC" ]; then
  echo ">> Compilando print.it-linux-amd64..."
  GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o "$BINARY_SRC" .
fi

rm -rf "$STAGE"
mkdir -p "$STAGE/DEBIAN"
mkdir -p "$STAGE/usr/bin"
mkdir -p "$STAGE/usr/lib/systemd/user"
mkdir -p "$STAGE/usr/share/print.it"

cp "$BINARY_SRC" "$STAGE/usr/bin/print.it"
chmod 755 "$STAGE/usr/bin/print.it"
cp packaging/linux/print.it.service "$STAGE/usr/lib/systemd/user/"
cp packaging/linux/uninstall.sh "$STAGE/usr/share/print.it/"
chmod 755 "$STAGE/usr/share/print.it/uninstall.sh"
cp packaging/linux/postinst "$STAGE/DEBIAN/"
cp packaging/linux/prerm "$STAGE/DEBIAN/"
chmod 755 "$STAGE/DEBIAN/postinst" "$STAGE/DEBIAN/prerm"

cat > "$STAGE/DEBIAN/control" <<EOF
Package: print-it
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: IdeYou <suporte@ideyou.com>
Description: print.it local printing agent
 Agente local de impressao termica ESC/POS para PDV.
EOF

dpkg-deb --root-owner-group --build "$STAGE" "$OUT"

echo ""
echo "Pacote gerado: $OUT"
