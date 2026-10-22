#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "=== Error Contract Bot ==="

# Backend error response shape
if ! grep -q '"code"' backend/internal/api/errors.go; then
  echo "ERROR: code field missing in errors.go"
  exit 1
fi
if ! grep -q '"message"' backend/internal/api/errors.go; then
  echo "ERROR: message field missing"
  exit 1
fi
if ! grep -q '"errors"' backend/internal/api/errors.go; then
  echo "ERROR: errors field missing"
  exit 1
fi

# Frontend client
if ! grep -q "code: string" frontend/src/lib/api/client.ts; then
  echo "WARNING: Frontend ApiError may not match contract"
fi

echo "Error contract OK"
