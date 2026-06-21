#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG="${ROOT}/packaging/build.config.json"
BUILD_JSON="${ROOT}/web/assets/build.json"
WEB_IMGS="${ROOT}/web/assets/imgs"

sync_app_icons() {
  mkdir -p "$WEB_IMGS"
  if [ -f "${ROOT}/packaging/appicon.png" ]; then
    cp "${ROOT}/packaging/appicon.png" "${WEB_IMGS}/appicon.png"
  fi
  if [ -f "${ROOT}/packaging/appicon.ico" ]; then
    cp "${ROOT}/packaging/appicon.ico" "${WEB_IMGS}/appicon.ico"
  fi
}

read_config() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$CONFIG" "$BUILD_JSON" <<'PY'
import json
import os
import sys

config_path, build_json_path = sys.argv[1:3]
root = os.path.dirname(os.path.dirname(config_path))
web_imgs = os.path.join(root, "web", "assets", "imgs")
os.makedirs(web_imgs, exist_ok=True)
for name in ("appicon.png", "appicon.ico"):
    src = os.path.join(root, "packaging", name)
    if os.path.isfile(src):
        with open(src, "rb") as icon_in:
            with open(os.path.join(web_imgs, name), "wb") as icon_out:
                icon_out.write(icon_in.read())

config = {}
if os.path.isfile(config_path):
    with open(config_path, encoding="utf-8") as f:
        config = json.load(f)

language = os.environ.get("PRINT_IT_LANGUAGE") or config.get("language") or "pt-br"
api_key = os.environ.get("PRINT_IT_BARCODES_API_KEY")
if api_key is None:
    api_key = config.get("barcodes_api_key") or ""

setup_lang = "en" if language == "en" else "pt"

os.makedirs(os.path.dirname(build_json_path), exist_ok=True)
with open(build_json_path, "w", encoding="utf-8") as f:
    json.dump({"language": language}, f, indent=2)
    f.write("\n")

def shell_quote(value: str) -> str:
    return "'" + value.replace("'", "'\"'\"'") + "'"

print(f"PRINT_IT_LANGUAGE={shell_quote(language)}")
print(f"PRINT_IT_BARCODES_API_KEY={shell_quote(api_key)}")
print(f"PRINT_IT_SETUP_LANG={shell_quote(setup_lang)}")
print(
    "PRINT_IT_LDFLAGS_BUILD="
    + shell_quote(
        f"-X main.buildBarcodesAPIKey={api_key} -X main.buildUILanguage={language}"
    )
)
PY
    return
  fi

  LANGUAGE="${PRINT_IT_LANGUAGE:-pt-br}"
  API_KEY="${PRINT_IT_BARCODES_API_KEY:-}"

  if [ -f "$CONFIG" ]; then
    LANGUAGE="$(sed -n 's/.*"language"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$CONFIG" | head -1)"
    [ -z "$LANGUAGE" ] && LANGUAGE="pt-br"
    if [ -z "${PRINT_IT_BARCODES_API_KEY:-}" ]; then
      API_KEY="$(sed -n 's/.*"barcodes_api_key"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$CONFIG" | head -1)"
    fi
  fi

  LANGUAGE="${PRINT_IT_LANGUAGE:-$LANGUAGE}"
  API_KEY="${PRINT_IT_BARCODES_API_KEY:-$API_KEY}"
  if [ "$LANGUAGE" = "en" ]; then
    SETUP_LANG="en"
  else
    SETUP_LANG="pt"
  fi

  mkdir -p "$(dirname "$BUILD_JSON")"
  sync_app_icons
  printf '{\n  "language": "%s"\n}\n' "$LANGUAGE" >"$BUILD_JSON"

  printf "PRINT_IT_LANGUAGE='%s'\n" "$LANGUAGE"
  printf "PRINT_IT_BARCODES_API_KEY='%s'\n" "$API_KEY"
  printf "PRINT_IT_SETUP_LANG='%s'\n" "$SETUP_LANG"
  printf "PRINT_IT_LDFLAGS_BUILD='-X main.buildBarcodesAPIKey=%s -X main.buildUILanguage=%s'\n" "$API_KEY" "$LANGUAGE"
}

case "${1:-}" in
  export)
    read_config
    ;;
  ldflags)
    eval "$(read_config)"
    printf '%s' "$PRINT_IT_LDFLAGS_BUILD"
    ;;
  setup-lang)
    eval "$(read_config)"
    printf '%s' "$PRINT_IT_SETUP_LANG"
    ;;
  *)
    eval "$(read_config)"
    ;;
esac
