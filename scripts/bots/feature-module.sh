#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

echo "=== Feature Module Bot ==="

FEATURES=$(find frontend/src/features -maxdepth 1 -mindepth 1 -type d)
for f in $FEATURES; do
  NAME=$(basename "$f")
  if [ ! -f "$f/index.ts" ]; then
    echo "ERROR: Feature $NAME missing index.ts"
    exit 1
  fi
  if [ ! -d "$f/components" ]; then
    echo "WARNING: Feature $NAME missing components/"
  fi
done
echo "Feature modules OK"
