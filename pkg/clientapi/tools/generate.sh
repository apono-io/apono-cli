#!/usr/bin/env bash

export OPENAPI_GENERATOR_VERSION=6.5.0

PROJECT_ROOT=$(dirname "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")")

echo "Project root: ${PROJECT_ROOT}"

echo "Removing previously generated files..."
xargs rm -v < .openapi-generator/FILES

echo "Generating client and model files..."
export GO_POST_PROCESS_FILE="gofmt -w"
exec openapi-generator-cli generate \
  --generator-name go \
  --input-spec api/openapi.json \
  --git-user-id apono-io \
  --git-repo-id apono-sdk-go \
  --output "$PROJECT_ROOT" \
  --config "$PROJECT_ROOT/config.yaml"
