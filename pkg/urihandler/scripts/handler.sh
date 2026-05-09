#!/bin/zsh
set -e
uri="$1"
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
if [[ "$session$account$client" == *%* ]]; then
  echo "URL-encoded characters not supported in launch params" >&2
  exit 64
fi
export _APONO_ACCOUNT_ID_="$account"
exec "__APONO_BINARY__" access use "$session" --client "$client" >/dev/null
