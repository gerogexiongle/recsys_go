#!/usr/bin/env bash
# Shared E2E helpers — single config: services/recommend/etc/recommend-api.yaml
# Toggle Kafka / rank LB via env (see config.ApplyEnvOverrides).
set -euo pipefail

RECSYS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RECSYS_CFG="${RECSYS_ROOT}/services/recommend/etc/recommend-api.yaml"
RECSYS_RANK_CFG="${RECSYS_ROOT}/services/rank/etc/rank-api.yaml"
RECSYS_E2E_HEALTH_RETRIES="${RECSYS_E2E_HEALTH_RETRIES:-80}"

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
    # fallback when fuser misses (other user / container)
    if command -v lsof >/dev/null 2>&1; then
      PIDS="$(lsof -ti ":${p}" 2>/dev/null || true)"
      if [[ -n "$PIDS" ]]; then
        kill -9 $PIDS 2>/dev/null || true
      fi
    fi
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

e2e_preflight_redis() {
  python3 - <<'PY'
import os, socket
host = os.environ.get("RECSYS_REDIS_HOST", "172.31.0.80")
port = int(os.environ.get("RECSYS_REDIS_PORT", "6379"))
s = socket.create_connection((host, port), 3)
s.close()
print(f"redis tcp ok {host}:{port}")
PY
}

e2e_diagnose_health_failure() {
  echo "FAIL: health timeout after ${RECSYS_E2E_HEALTH_RETRIES} retries" >&2
  echo "  rank pid=${RANK_PID:-?} recommend pid=${REC_PID:-?}" >&2
  for p in 18080 18081; do
    if ss -tln 2>/dev/null | grep -q ":${p} "; then
      echo "  port ${p}: listening" >&2
    else
      echo "  port ${p}: not listening" >&2
    fi
  done
  if [[ -n "${RANK_LOG:-}" && -f "$RANK_LOG" ]]; then
    echo "----- tail rank log ($RANK_LOG) -----" >&2
    tail -40 "$RANK_LOG" >&2 || true
  fi
  if [[ -n "${REC_LOG:-}" && -f "$REC_LOG" ]]; then
    echo "----- tail recommend log ($REC_LOG) -----" >&2
    tail -40 "$REC_LOG" >&2 || true
  fi
}

e2e_curl_health() {
  local url="$1"
  curl -sf -m 2 "$url" >/dev/null 2>&1
}

e2e_wait_health() {
  local n=0
  while [[ "$n" -lt "$RECSYS_E2E_HEALTH_RETRIES" ]]; do
    if e2e_curl_health "http://127.0.0.1:18081/health" && \
       e2e_curl_health "http://127.0.0.1:18080/health"; then
      return 0
    fi
    # process died early → fail fast with logs
    if [[ -n "${RANK_PID:-}" ]] && ! kill -0 "$RANK_PID" 2>/dev/null; then
      echo "FAIL: rank-api exited before health (pid=$RANK_PID)" >&2
      e2e_diagnose_health_failure
      return 1
    fi
    if [[ -n "${REC_PID:-}" ]] && ! kill -0 "$REC_PID" 2>/dev/null; then
      echo "FAIL: recommend-api exited before health (pid=$REC_PID)" >&2
      e2e_diagnose_health_failure
      return 1
    fi
    n=$((n + 1))
    sleep 0.25
  done
  e2e_diagnose_health_failure
  return 1
}

# Start rank first (FM load + redis), then recommend.
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
  : >"$RANK_LOG"
  : >"$REC_LOG"

  ./bin/rank-api -f "$RECSYS_RANK_CFG" >>"$RANK_LOG" 2>&1 &
  RANK_PID=$!

  local n=0
  while [[ "$n" -lt "$RECSYS_E2E_HEALTH_RETRIES" ]]; do
    if e2e_curl_health "http://127.0.0.1:18081/health"; then
      break
    fi
    if ! kill -0 "$RANK_PID" 2>/dev/null; then
      echo "FAIL: rank-api exited during startup" >&2
      e2e_diagnose_health_failure
      return 1
    fi
    n=$((n + 1))
    sleep 0.25
  done
  if ! e2e_curl_health "http://127.0.0.1:18081/health"; then
    echo "FAIL: rank-api health timeout" >&2
    e2e_diagnose_health_failure
    return 1
  fi

  ./bin/recommend-api -f "$RECSYS_CFG" >>"$REC_LOG" 2>&1 &
  REC_PID=$!
  export RANK_LOG REC_LOG
}

e2e_stop_services() {
  kill "${REC_PID:-}" "${RANK_PID:-}" 2>/dev/null || true
  wait "${REC_PID:-}" "${RANK_PID:-}" 2>/dev/null || true
  REC_PID= RANK_PID=
}
