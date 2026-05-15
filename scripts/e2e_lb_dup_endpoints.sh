#!/usr/bin/env bash
# Rank LB-only slice of full E2E (same recommend-api.yaml + RECSYS_RANK_ENDPOINTS).
set -euo pipefail
export RECSYS_E2E_PHASES=lb
exec "$(cd "$(dirname "$0")/.." && pwd)/scripts/e2e_full_chain.sh"
