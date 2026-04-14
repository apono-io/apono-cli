# AponoCLI Development Guide

This guide covers day-to-day development workflows for the AponoCLI.

## Login / Profile Setup

The CLI supports multiple profiles so you can switch between environments. Each profile stores its own tokens and API URLs in `~/.config/apono-cli/config.json`.

### Local

```bash
apono login --p local \
  --api-url http://localhost:8090 \
  --app-url http://localhost:9000 \
  --portal-url http://localhost:9010 \
  --token-url http://localhost:8095
```

### Production

```bash
apono login --p prod
```

Production uses the default URLs (no flags needed).

### Switching profiles

```bash
apono profiles get-profiles      # list all profiles
apono profiles set-profile prod  # set active profile
apono --profile local <command>  # one-off override
```

## Build and Run

```bash
go build         # produces ./apono binary in the project root
./apono --help
```

Every code change must be rebuilt to be reflected in the binary.

## Testing Local Builds Against Any Environment

The local build and the profile system are independent — you can run your locally-built `./apono` binary against any profile you've set up.

```bash
# Run local build against local backend
./apono --profile local requests list

# Run local build against prod
./apono --profile prod requests list
```

This is useful for testing a feature branch against a specific environment without needing to deploy the CLI.

## Config File

Profiles, tokens, and settings are persisted to:

```
~/.config/apono-cli/config.json
```

Useful when debugging auth issues — you can inspect the file directly to see which profiles exist, their URLs, and token state. Deleting the file resets all profiles.

## OpenAPI Client Generation

The API client models (`pkg/clientapi/`) are generated from an OpenAPI spec located at:

```
pkg/clientapi/api/openapi.json
```

This file is **copied from the `apono-mono` project** at `openapi/client-api.json` (not to be confused with `admin-api.json`, `public-api.json`, or the other specs in that directory).

When the server's client API changes:

1. Copy the updated spec: `cp <path-to-apono-mono>/openapi/client-api.json pkg/clientapi/api/openapi.json`
2. Regenerate the client: `make gen`
