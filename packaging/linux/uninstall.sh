#!/usr/bin/env bash
set -euo pipefail

if command -v systemctl >/dev/null 2>&1; then
  systemctl --user disable --now print.it.service 2>/dev/null || true
  rm -f "$HOME/.config/systemd/user/default.target.wants/print.it.service"
fi

if [ "$(id -u)" -eq 0 ]; then
  apt-get remove -y print-it 2>/dev/null || dpkg -r print-it 2>/dev/null || true
  rm -f /usr/bin/print.it
  rm -f /usr/lib/systemd/user/print.it.service
  rm -rf /usr/share/print.it
else
  echo "Execute com sudo para desinstalar completamente, ou use o gerenciador de pacotes."
fi

echo "print.it desinstalado."
