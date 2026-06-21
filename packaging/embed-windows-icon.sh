#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ ! -f packaging/appicon.png ]; then
  echo "packaging/appicon.png nao encontrado" >&2
  exit 1
fi

go run github.com/tc-hib/go-winres@v0.3.3 make --in packaging/windows/winres.json --out .
