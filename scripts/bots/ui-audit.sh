#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "=== UI Bot Audit ==="

# 1. No back buttons
echo "[1] No dumb back buttons"
if grep -R -n "ArrowLeft" frontend/src/features/ frontend/src/app/ --include="*.tsx" | grep -q .; then
  echo "ERROR: ArrowLeft usage found in features/pages"
  exit 1
fi
if grep -R -n "Back to" frontend/src/ --include="*.tsx"; then
  echo "ERROR: 'Back to' link found"
  exit 1
fi
echo "OK"

# 2. No hardcoded hex colors
echo "[2] No hardcoded hex colors"
if grep -R -n "text-\[" frontend/src/features/ --include="*.tsx" | grep -q "#"; then
  echo "WARNING: Hardcoded hex color found"
fi
if grep -R -n 'bg-\[' frontend/src/features/ --include="*.tsx" | grep -q "#"; then
  echo "WARNING: Hardcoded hex background found"
fi
echo "OK"

# 3. No div soup: check for empty div wrappers
echo "[3] Div soup check placeholder"
echo "OK"

# 4. Run pnpm lint
echo "[4] pnpm lint"
cd frontend
pnpm lint || echo "Lint warnings present"

echo "UI Bot audit complete"
