#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"
BINARY="$ROOT/dist/print.it-windows-amd64.exe"

if [ ! -f "$BINARY" ]; then
  echo ">> Compilando print.it-windows-amd64.exe (CGO)..."
  if [ -z "${MSYSTEM:-}" ] && [ "$(uname -s)" != "MINGW"* ] && [ "$(uname -s)" != "MSYS"* ]; then
    echo "Build Windows requer MSYS2/MinGW com CGO_ENABLED=1." >&2
    echo "Use GitHub Actions ou: msys2 -> pacman -S mingw-w64-x86_64-gcc -> CGO_ENABLED=1 go build ..." >&2
    exit 1
  fi
  eval "$(packaging/read-build-config.sh export)"
  CGO_ENABLED=1 go build -ldflags "-s -w -H=windowsgui ${PRINT_IT_LDFLAGS_BUILD}" -o "$BINARY" .
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

SETUP_LANG="$(packaging/read-build-config.sh setup-lang)"

"$ISCC" "//DMyAppVersion=$VERSION" "//DSetupLanguage=$SETUP_LANG" packaging/windows/printit.iss

echo ""
echo "Instalador gerado em dist/print.it-${VERSION}-windows-amd64.exe"
