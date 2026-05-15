#!/usr/bin/env bash
# Kafka-only slice of full E2E (same recommend-api.yaml + RECSYS_KAFKA_PUSH=1).
set -euo pipefail
export RECSYS_E2E_PHASES=kafka
exec "$(cd "$(dirname "$0")/.." && pwd)/scripts/e2e_full_chain.sh"
