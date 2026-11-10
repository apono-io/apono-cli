#!/bin/zsh
set -e
zmodload zsh/datetime 2>/dev/null || true

invocation="$$-${EPOCHSECONDS:-$(date +%s)}"
logfile="${HOME}/Library/Application Support/apono-cli/handler.log"

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

safe_token() {
  if [[ "$1" =~ "$2" ]]; then
    print -r -- "$1"
  else
    print -r -- "<redacted>"
  fi
}

describe_target() {
  local t="$1" scheme rest authority host has_userinfo=no has_path=no
  if [[ "$t" != *://* ]]; then
    print -r -- "target=<redacted> has_userinfo=no has_path=no"
    return
  fi
  scheme="${t%%://*}"
  rest="${t#*://}"
  authority="${rest%%/*}"
  [[ "$rest" == */* ]] && has_path=yes
  host="$authority"
  if [[ "$authority" == *@* ]]; then
    has_userinfo=yes
    host="${authority##*@}"
  fi
  print -r -- "target=$(safe_token "$scheme" '^[A-Za-z][A-Za-z0-9+.-]{0,15}$')://$(safe_token "$host" '^[A-Za-z0-9.-]{1,64}$') has_userinfo=$has_userinfo has_path=$has_path"
}

describe_uri() {
  local u="$1" query="" kv name entries=()
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
      *) entries+=("$(safe_token "$name" '^[A-Za-z0-9_-]{1,32}$')=<redacted>") ;;
    esac
  done
  print -r -- "$(describe_target "${u%%\?*}") params=[${(j:,:)entries}]"
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
