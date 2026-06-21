#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(sed -n 's/^const version = "\(.*\)"/\1/p' version.go)"

if ! command -v go >/dev/null 2>&1; then
  echo "Go nao encontrado." >&2
  exit 1
fi

echo ">> print.it $VERSION"
echo ">> Baixando dependencias..."
go mod tidy

mkdir -p dist

build_one() {
  local goos="$1"
  local goarch="$2"
  local out="$3"
  echo ">> GOOS=$goos GOARCH=$goarch -> $out"
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "-s -w" -o "$out" .
}

build_one darwin arm64 "dist/print.it-darwin-arm64"
build_one darwin amd64 "dist/print.it-darwin-amd64"
build_one linux amd64 "dist/print.it-linux-amd64"
echo ">> Windows: compile no runner windows-latest (CGO) ou MSYS2 local"

echo ""
echo "Binarios em dist/:"
ls -lh dist/print.it-*

if [ "$(uname -s)" = "Darwin" ]; then
  echo ""
  echo ">> Empacotando macOS (.pkg)..."
  chmod +x packaging/macos/build-pkg.sh
  packaging/macos/build-pkg.sh
fi

if [ "$(uname -s)" = "Linux" ] && command -v dpkg-deb >/dev/null 2>&1; then
  echo ""
  echo ">> Empacotando Linux (.deb)..."
  chmod +x packaging/linux/build-deb.sh
  packaging/linux/build-deb.sh
fi

if [ "$(uname -s)" = "MINGW"* ] || [ "$(uname -s)" = "MSYS"* ] || command -v iscc >/dev/null 2>&1; then
  if command -v iscc >/dev/null 2>&1; then
    echo ""
    echo ">> Empacotando Windows (Inno Setup)..."
    chmod +x packaging/windows/build-msi.sh
    packaging/windows/build-msi.sh
  fi
fi

echo ""
echo "Pronto."
