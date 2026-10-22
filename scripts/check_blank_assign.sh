#!/usr/bin/env bash
set -euo pipefail

echo "Checking for blank assignments '_' in production Go code..."

# Find _ = patterns excluding tests
matches=$(grep -r --include="*.go" -n "_ = " backend/internal/worker backend/internal/services backend/internal/api | grep -v "_test.go" || true)

if [[ -n "$matches" ]]; then
  echo "Found blank assignments:"
  echo "$matches"
  echo ""
  echo "Fail: Silent error discards detected. Fix or log them."
  exit 1
else
  echo "OK: No blank assignments found."
fi
