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
  chmod +x packaging/embed-windows-icon.sh
  packaging/embed-windows-icon.sh
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

cp packaging/appicon.ico packaging/windows/appicon.ico
cp packaging/delicon.ico packaging/windows/delicon.ico

if python3 -c "from PIL import Image" 2>/dev/null; then
  python3 - <<'PY'
from PIL import Image

src = Image.open("packaging/appicon.png").convert("RGBA")
size = 55
padding = 10
inner = size - 2 * padding
w, h = src.size
scale = min(inner / w, inner / h)
nw, nh = max(1, int(w * scale)), max(1, int(h * scale))
resized = src.resize((nw, nh), Image.LANCZOS)
canvas = Image.new("RGBA", (size, size), (255, 255, 255, 0))
canvas.paste(resized, ((size - nw) // 2, (size - nh) // 2), resized)
canvas.save("packaging/windows/wizard-icon.png")
PY
elif [ ! -f packaging/windows/wizard-icon.png ]; then
  echo "wizard-icon.png ausente e Pillow indisponivel para gerar" >&2
  exit 1
fi

"$ISCC" "//DMyAppVersion=$VERSION" "//DSetupLanguage=$SETUP_LANG" packaging/windows/printit.iss

echo ""
echo "Instalador gerado em dist/print.it-${VERSION}-windows-amd64.exe"
