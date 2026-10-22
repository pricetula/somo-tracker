#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "=== Security Bot ==="
echo "Checking for BOLA patterns..."

# Check handlers for userID from context
if ! grep -R "c.Locals(\"userID\")" backend/internal/api/ --include="*.go" | grep -q .; then
  echo "WARNING: No c.Locals(\"userID\") usage found in handlers"
fi

# Check error contract usage
if ! grep -R "WriteError" backend/internal/api/ --include="*.go" | grep -q .; then
  echo "WARNING: WriteError not used"
fi

# Check for raw sqlc usage in handlers
if grep -R "sqlc\." backend/internal/api/ --include="*.go"; then
  echo "ERROR: Direct sqlc usage in handlers detected"
  exit 1
fi

echo "Security bot checks complete"
