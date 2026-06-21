#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -f packaging/appicon.png ]; then
  echo "packaging/appicon.png nao encontrado" >&2
  exit 1
fi

if python3 -c "from PIL import Image" 2>/dev/null; then
  python3 - <<'PY'
from PIL import Image
from pathlib import Path

root = Path("packaging")
sizes = [(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)]

def save_ico(src_path: Path, dst_path: Path) -> None:
    src = Image.open(src_path).convert("RGBA")
    src.save(dst_path, format="ICO", sizes=sizes)

save_ico(root / "appicon.png", root / "appicon.ico")
if (root / "delicon.ico").is_file():
    del_src = Image.open(root / "delicon.ico").convert("RGBA")
    del_src.save(root / "delicon.ico", format="ICO", sizes=sizes)
PY
elif [ ! -f packaging/appicon.ico ]; then
  echo "packaging/appicon.ico ausente e Pillow indisponivel para gerar" >&2
  exit 1
fi

if [ ! -f packaging/appicon.ico ]; then
  echo "packaging/appicon.ico nao encontrado" >&2
  exit 1
fi

WIN_DIR="$ROOT/packaging/windows"
cp packaging/appicon.ico "$WIN_DIR/appicon.ico"

if [ "${1:-}" = "icons-only" ]; then
  exit 0
fi

SYSO="$ROOT/rsrc_windows_amd64.syso"

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
