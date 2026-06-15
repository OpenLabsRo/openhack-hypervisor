#!/usr/bin/env bash
set -euo pipefail

if ! command -v swag >/dev/null 2>&1; then
  echo "swag CLI not found. Install with 'go install github.com/swaggo/swag/cmd/swag@latest'" >&2
  exit 1
fi

swag init -g cmd/server/main.go -o internal/swagger/docs --parseInternal "$@"

