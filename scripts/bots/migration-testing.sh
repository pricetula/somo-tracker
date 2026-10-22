#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "=== Migration Testing Bot ==="

MIGRATIONS=$(find backend/db/migrations -type f -name "*.sql" | sort)
COUNT=0
for f in $MIGRATIONS; do
  NAME=$(basename "$f" .sql)
  # Expect test function
  if ! grep -rq "TestMigrator.*$NAME" backend/internal/database/; then
    echo "WARNING: No test for migration $NAME"
  else
    echo "OK: $NAME"
  fi
  COUNT=$((COUNT+1))
done
echo "Checked $COUNT migrations"
