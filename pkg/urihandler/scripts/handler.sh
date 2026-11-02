#!/bin/zsh
set -e

logfile="${HOME}/Library/Application Support/apono-cli/handler.log"
log() { print -r -- "$1"$'\t'"$2" >> "$logfile" 2>/dev/null || true }
describe_path() {
  local brew=no userbin=no
  case ":$PATH:" in
    *":/opt/homebrew/bin:"*|*":/usr/local/bin:"*) brew=yes ;;
  esac
  case ":$PATH:" in
    *":$HOME/.local/bin:"*|*":$HOME/bin:"*) userbin=yes ;;
  esac
  print -r -- "brew=$brew userbin=$userbin"
}

trap '
  code=$?
  if [[ $code -ne 0 ]]; then
    case $code in
      64)  level=ERROR; reason="invalid launch URL" ;;
      127) level=WARN;  reason="apono CLI not found on PATH" ;;
      *)   level=WARN;  reason="handler failed" ;;
    esac
    log "$level" "$reason code=$code $(describe_path)"
  fi
' EXIT

uri="$1"
log INFO "received launch request"
if [[ -z "$uri" ]]; then
  echo "missing URI argument" >&2
  exit 64
fi
if [[ "$uri" != apono://connect\?* ]]; then
  echo "unsupported URI: $uri" >&2
  exit 64
fi
query="${uri#*\?}"
session=""; account=""; client=""
for kv in ${(s:&:)query}; do
  case "$kv" in
    session=*) session="${kv#session=}" ;;
    account=*) account="${kv#account=}" ;;
    client=*)  client="${kv#client=}" ;;
  esac
done
if [[ -z "$session" || -z "$account" || -z "$client" ]]; then
  echo "missing required params in: $uri" >&2
  exit 64
fi
# Session IDs may carry percent-encoded characters (e.g. %2F from a Secrets
# Manager path). Pass them through still encoded; the CLI decodes exactly once.
if [[ "$account$client" == *%* ]]; then
  echo "URL-encoded characters not supported in account/client params" >&2
  exit 64
fi
session_has_pct=no
[[ "$session" == *%* ]] && session_has_pct=yes
log INFO "parsed launch params session=$session account=$account client=$client session_has_pct=$session_has_pct"
if ! command -v apono >/dev/null 2>&1; then
  echo "apono CLI not found on PATH" >&2
  exit 127
fi
log INFO "apono resolved; handing off to access use"
export _APONO_ACCOUNT_ID_="$account"
export _APONO_SESSION_URL_ENCODED_=1
exec apono access use --client "$client" -- "$session" >/dev/null
