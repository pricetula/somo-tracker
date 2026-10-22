#!/usr/bin/env bash
set -euo pipefail

echo "=== Bot Factory Run All ==="
bash scripts/bots/error-contract.sh
bash scripts/bots/api-audit.sh || true
bash scripts/bots/ui-audit.sh || true
bash scripts/bots/security-audit.sh || true
bash scripts/bots/migration-testing.sh || true
bash scripts/bots/feature-module.sh || true
echo "All bots completed"
