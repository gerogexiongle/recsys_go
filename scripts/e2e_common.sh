#!/usr/bin/env bash
# Shared E2E helpers — single config: services/recommend/etc/recommend-api.yaml
# Toggle Kafka / rank LB via env (see config.ApplyEnvOverrides).
set -euo pipefail

RECSYS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RECSYS_CFG="${RECSYS_ROOT}/services/recommend/etc/recommend-api.yaml"
RECSYS_RANK_CFG="${RECSYS_ROOT}/services/rank/etc/rank-api.yaml"

e2e_export_defaults() {
  export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
  export RECSYS_SEED_REDIS="${RECSYS_SEED_REDIS:-1}"
  export RECSYS_REDIS_HOST="${RECSYS_REDIS_HOST:-172.31.0.80}"
  export RECSYS_REDIS_PORT="${RECSYS_REDIS_PORT:-6379}"
  export RECSYS_REDIS_CRYPTO="${RECSYS_REDIS_CRYPTO:-1}"
  export RECSYS_REDIS_PASSWORD_HEX="${RECSYS_REDIS_PASSWORD_HEX:-d1c98bea6a9824201ac9375488748b3c07}"
  export KAFKA_BROKERS="${KAFKA_BROKERS:-127.0.0.1:9092}"
  export KAFKA_TOPIC="${KAFKA_TOPIC:-test}"
}

e2e_free_ports() {
  for p in 18080 18081; do
    fuser -k "${p}/tcp" >/dev/null 2>&1 || true
  done
  sleep 1
}

e2e_build() {
  cd "$RECSYS_ROOT"
  go build -o bin/rank-api ./services/rank
  go build -o bin/recommend-api ./services/recommend
}

e2e_seed_redis() {
  cd "$RECSYS_ROOT"
  python3 scripts/seed_feature_redis.py
}

e2e_wait_health() {
  for _ in $(seq 1 40); do
    if curl -sf -m 1 http://127.0.0.1:18081/health >/dev/null && \
       curl -sf -m 1 http://127.0.0.1:18080/health >/dev/null; then
      return 0
    fi
    sleep 0.25
  done
  echo "FAIL: health timeout" >&2
  return 1
}

# start_rank + start_recommend; sets RANK_PID REC_PID. Caller must trap kill.
e2e_start_services() {
  unset RECSYS_KAFKA_PUSH RECSYS_KAFKA_BROKERS RECSYS_KAFKA_TOPIC RECSYS_RANK_ENDPOINTS
  if [[ "${RECSYS_E2E_KAFKA:-}" == "1" ]]; then
    export RECSYS_KAFKA_PUSH=1
    export RECSYS_KAFKA_BROKERS="$KAFKA_BROKERS"
    export RECSYS_KAFKA_TOPIC="$KAFKA_TOPIC"
  fi
  if [[ -n "${RECSYS_E2E_RANK_ENDPOINTS:-}" ]]; then
    export RECSYS_RANK_ENDPOINTS
  fi

  cd "$RECSYS_ROOT"
  RANK_LOG="${RECSYS_E2E_RANK_LOG:-${TMPDIR:-/tmp}/recsys_e2e_rank.log}"
  REC_LOG="${RECSYS_E2E_REC_LOG:-${TMPDIR:-/tmp}/recsys_e2e_rec.log}"
  ./bin/rank-api -f "$RECSYS_RANK_CFG" >>"$RANK_LOG" 2>&1 &
  RANK_PID=$!
  ./bin/recommend-api -f "$RECSYS_CFG" >>"$REC_LOG" 2>&1 &
  REC_PID=$!
  export RANK_LOG REC_LOG
}

e2e_stop_services() {
  kill "${REC_PID:-}" "${RANK_PID:-}" 2>/dev/null || true
  wait "${REC_PID:-}" "${RANK_PID:-}" 2>/dev/null || true
}
