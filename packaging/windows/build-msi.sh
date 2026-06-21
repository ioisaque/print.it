#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"
BINARY="$ROOT/dist/print.it-windows-amd64.exe"

if [ ! -f "$BINARY" ]; then
  echo ">> Compilando print.it-windows-amd64.exe..."
  GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o "$BINARY" .
fi

if command -v iscc >/dev/null 2>&1; then
  ISCC="iscc"
elif [ -f "/c/Program Files (x86)/Inno Setup 6/ISCC.exe" ]; then
  ISCC="/c/Program Files (x86)/Inno Setup 6/ISCC.exe"
elif [ -f "/c/Program Files/Inno Setup 6/ISCC.exe" ]; then
  ISCC="/c/Program Files/Inno Setup 6/ISCC.exe"
else
  echo "Inno Setup (iscc) nao encontrado." >&2
  exit 1
fi

"$ISCC" "/DMyAppVersion=$VERSION" packaging/windows/printit.iss

echo ""
echo "Instalador gerado em dist/print.it-${VERSION}-windows-amd64.exe"
