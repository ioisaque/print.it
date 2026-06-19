#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  echo "Go nao encontrado. Instale com: brew install go"
  exit 1
fi

echo ">> Baixando dependencias..."
go mod tidy

echo ">> Compilando..."
go build -o print.it .

if [ ! -f config.json ]; then
  cp config.example.json config.json
  echo ">> config.json criado a partir do exemplo."
  echo "   Edite printer_host com o IP da sua impressora."
fi

echo ""
echo "Pronto! Para iniciar:"
echo "  ./print.it"
echo ""
echo "Teste rapido (em outro terminal):"
echo "  curl http://127.0.0.1:9280/health"
echo "  curl -X POST http://127.0.0.1:9280/print/test"
