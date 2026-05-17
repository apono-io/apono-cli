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
# zsh -l sources .zprofile but not .zshrc, so keg-only brew formulas
# (mysql-client, postgresql@*, etc.) miss PATH even when the user's
# Terminal sees them. Append every brew opt/*/bin to fill the gap.
for opt_dir in /opt/homebrew/opt/*/bin(N/) /usr/local/opt/*/bin(N/); do
  PATH="$PATH:$opt_dir"
done
export PATH

export _APONO_ACCOUNT_ID_="$account"
exec apono access use "$session" --client "$client" >/dev/null
