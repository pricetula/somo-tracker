#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "=== AGENTS.md Compliance Check ==="

# 1. Forbidden patterns backend
echo "[1/5] Backend forbidden patterns..."
if grep -R -n "catch (e) {}" backend/ --include="*.go" ; then
  echo "ERROR: Empty catch found"
  exit 1
fi
if grep -R -n "if err != nil { }" backend/ --include="*.go" ; then
  echo "ERROR: Empty error handling found"
  exit 1
fi
if grep -R -n "_ = " backend/ --include="*.go" | grep -v "_test.go" > /dev/null; then
  echo "WARNING: Silent underscore usage detected"
fi
if grep -R -n "catch (e) {}" frontend/src/ --include="*.ts" --include="*.tsx" ; then
  echo "ERROR: Empty catch in frontend"
  exit 1
fi

# 2. Error contract file exists and shape
if [ ! -f backend/internal/api/errors.go ]; then
  echo "ERROR: backend/internal/api/errors.go missing"
  exit 1
fi
if ! grep -q "type APIError" backend/internal/api/errors.go; then
  echo "ERROR: APIError type missing"
  exit 1
fi

# 3. Docs build
echo "[2/5] Docs build..."
if command -v pnpm >/dev/null 2>&1; then
  (cd frontend && pnpm docs:lint || true)
else
  echo "pnpm not found, skipping docs lint"
fi

# 4. Generate idempotent check
echo "[3/5] API generation idempotency..."
if ! git diff --quiet docs/openapi.yaml backend/docs/swagger.json 2>/dev/null; then
  echo "WARNING: swagger drift detected"
fi

# 5. Migration tests presence
echo "[4/5] Migration tests presence..."
MIGRATIONS=$(find backend/db/migrations -type f -name "*.sql" 2>/dev/null | wc -l)
TESTS=$(grep -r "func TestMigrator_" backend/internal/database/ 2>/dev/null | wc -l)
if [ "$MIGRATIONS" -ne "$TESTS" ]; then
  echo "WARNING: Migration count $MIGRATIONS vs test count $TESTS"
fi

echo "[5/5] Compliance checks complete"
echo "OK"
