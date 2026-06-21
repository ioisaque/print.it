#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -f packaging/appicon.ico ]; then
  echo "packaging/appicon.ico nao encontrado" >&2
  exit 1
fi

WIN_DIR="$ROOT/packaging/windows"
SYSO="$ROOT/rsrc_windows_amd64.syso"

cp packaging/appicon.ico "$WIN_DIR/appicon.ico"

windres_cmd=""
if command -v windres >/dev/null 2>&1; then
  windres_cmd="windres"
elif command -v x86_64-w64-mingw32-windres >/dev/null 2>&1; then
  windres_cmd="x86_64-w64-mingw32-windres"
else
  echo "windres nao encontrado (instale mingw-w64-x86_64-gcc)" >&2
  exit 1
fi

rm -f "$SYSO"
(
  cd "$WIN_DIR"
  "$windres_cmd" --target=pe-x86-64 -O coff -o "$SYSO" icon.rc
)

if [ ! -f "$SYSO" ]; then
  echo "falha ao gerar $SYSO" >&2
  exit 1
fi
