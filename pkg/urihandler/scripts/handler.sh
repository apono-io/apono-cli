#!/bin/zsh
set -e
zmodload zsh/datetime 2>/dev/null || true

invocation="$$-${EPOCHSECONDS:-$(date +%s)}"
logfile="${HOME}/Library/Application Support/apono-cli/handler.log"
# Lines are shipped later, by the next `apono access use`, so each one records
# when it happened (at=, epoch seconds) and which click wrote it (inv=).
log() { print -r -- "$1"$'\t'"$2 at=${EPOCHREALTIME:-$(date +%s)} inv=$invocation" >> "$logfile" 2>/dev/null || true }
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
# clip strips control characters (keeps one log line per event) and caps length.
clip() {
  local v="${1//[[:cntrl:]]/}"
  print -r -- "${v[1,$2]}"
}
# describe_uri logs the shape of the launch URL: target, parameter names, and
# values only for the known launch parameters. Unknown parameter values are
# never logged.
describe_uri() {
  local u="$1" target query="" kv name entries=()
  target="${u%%\?*}"
  [[ "$u" == *\?* ]] && query="${u#*\?}"
  for kv in ${(s:&:)query}; do
    name="${kv%%=*}"
    case "$name" in
      session|account|client)
        if [[ "$kv" == *=* ]]; then
          entries+=("$name=$(clip "${kv#*=}" 200)")
        else
          entries+=("$name")
        fi
        ;;
      *) entries+=("$(clip "$name" 64)=<redacted>") ;;
    esac
  done
  print -r -- "target=$(clip "$target" 64) params=[${(j:,:)entries}]"
}

fail_reason=unknown
trap '
  code=$?
  if [[ $code -ne 0 ]]; then
    case $code in
      64)  level=ERROR; reason="invalid launch URL"; detail="reason=$fail_reason $(describe_uri "$uri") " ;;
      127) level=WARN;  reason="apono CLI not found on PATH"; detail="" ;;
      *)   level=WARN;  reason="handler failed"; detail="" ;;
    esac
    log "$level" "$reason code=$code $detail$(describe_path)"
  fi
' EXIT

uri="$1"
log INFO "received launch request"
if [[ -z "$uri" ]]; then
  echo "missing URI argument" >&2
  fail_reason=missing_uri
  exit 64
fi
if [[ "$uri" != apono://connect\?* ]]; then
  echo "unsupported URI: $uri" >&2
  fail_reason=unsupported_target
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
  fail_reason=missing_params
  exit 64
fi
if [[ "$account$client" == *%* ]]; then
  echo "URL-encoded characters not supported in account/client params" >&2
  fail_reason=encoded_account_client
  exit 64
fi
session_has_pct=no
[[ "$session" == *%* ]] && session_has_pct=yes
log INFO "parsed launch params session=$(clip "$session" 200) account=$(clip "$account" 200) client=$(clip "$client" 200) session_has_pct=$session_has_pct"
if ! command -v apono >/dev/null 2>&1; then
  echo "apono CLI not found on PATH" >&2
  exit 127
fi
log INFO "apono resolved; handing off to access use"
export _APONO_ACCOUNT_ID_="$account"
export _APONO_SESSION_URL_ENCODED_=1
exec apono access use --client "$client" -- "$session" >/dev/null
