#!/usr/bin/env bash
# Full E2E: one recommend-api.yaml + env overrides — center pipeline, Kafka audit log, rank LB.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck source=scripts/e2e_common.sh
source scripts/e2e_common.sh

e2e_export_defaults
PHASES="${RECSYS_E2E_PHASES:-center,kafka,lb}"

echo "==> Config: ${RECSYS_CFG} (env: RECSYS_KAFKA_PUSH, RECSYS_RANK_ENDPOINTS)"
echo "==> Phases: ${PHASES}"

echo "==> Seed Redis at ${RECSYS_REDIS_HOST}:${RECSYS_REDIS_PORT}"
e2e_seed_redis
echo "==> Preflight Redis TCP"
e2e_preflight_redis || { echo "FAIL: cannot reach Redis; export RECSYS_REDIS_HOST"; exit 1; }
e2e_free_ports
echo "==> Build"
e2e_build

cleanup() { e2e_stop_services; }
trap cleanup EXIT

# --- Phase 1: center (recall / merge / filter / rank / show) ---
if [[ ",${PHASES}," == *",center,"* ]]; then
  echo ""
  echo "========== PHASE center =========="
  e2e_start_services
  e2e_wait_health
  curl -sf -m 2 http://127.0.0.1:18081/health | grep -qx ok
  curl -sf -m 2 http://127.0.0.1:18080/health | grep -qx ok
  READY="$(curl -sf -m 2 http://127.0.0.1:18080/v1/ready)"
  echo "ready: $READY"
  echo "$READY" | grep -q '"center":true' || { echo "FAIL: center config"; tail -30 "$REC_LOG"; exit 1; }

  OUT="$(curl -sf -m 15 -X POST http://127.0.0.1:18080/v1/recommend \
    -H 'Content-Type: application/json' \
    -d '{"uuid":"full-e2e-center","user_id":900001,"exp_ids":[0],"ret_count":5}')"
  echo "$OUT"

  python3 - <<'PY' "$OUT"
import json, sys
resp = json.loads(sys.argv[1])
ids = resp.get("item_ids") or []
print("item_ids:", ids)
assert 910005 not in ids, "910005 filtered by LiveExposure"
assert 910009 not in ids, "910009 dropped (no item portrait)"
assert 910001 in ids, "910001 LiveRedirect"
assert len(ids) >= 3
recall_types = {r.get("recall_type") for r in resp.get("recall") or []}
assert "CrossTag7d" in recall_types, recall_types
cross_ids = [r["item_id"] for r in (resp.get("recall") or []) if r.get("recall_type") == "CrossTag7d"]
assert any(i in cross_ids for i in (910006, 910007, 910008)), cross_ids
assert ids[0] == 910001, f"ForcedInsert expects 910001 first, got {ids[0]}"
if 910010 in ids and 910004 in ids:
    assert ids.index(910010) < ids.index(910004), "FM rank order"
print("CENTER PHASE OK")
PY
  e2e_stop_services
  e2e_free_ports
fi

# --- Phase 2: Kafka algorithm log ---
if [[ ",${PHASES}," == *",kafka,"* ]]; then
  echo ""
  echo "========== PHASE kafka =========="
  export RECSYS_E2E_KAFKA=1
  RECSYS_E2E_REC_LOG="${TMPDIR:-/tmp}/recsys_kafka_rec.log"
  e2e_start_services
  e2e_wait_health
  curl -sf http://127.0.0.1:18080/v1/ready | grep -q '"center":true'

  REQ_UUID="kafka-e2e-$(date +%s)"
  echo "==> POST /v1/recommend uuid=${REQ_UUID}"
  OUT="$(curl -sf -m 15 -X POST http://127.0.0.1:18080/v1/recommend \
    -H 'Content-Type: application/json' \
    -d "{\"uuid\":\"${REQ_UUID}\",\"user_id\":900001,\"exp_ids\":[0],\"ret_count\":5}")"
  echo "Response: $OUT"

  KAFKA_MSG="${TMPDIR:-/tmp}/recsys_kafka_msg.txt"
  sleep 4
  if ! go run scripts/kafka_consume_latest.go -brokers "$KAFKA_BROKERS" -topic "$KAFKA_TOPIC" \
      -match "$REQ_UUID" -timeout 20s >"$KAFKA_MSG" 2>/dev/null; then
    sleep 3
    go run scripts/kafka_consume_latest.go -brokers "$KAFKA_BROKERS" -topic "$KAFKA_TOPIC" \
      -match "$REQ_UUID" -timeout 15s >"$KAFKA_MSG" 2>/dev/null || true
  fi
  if [[ ! -s "$KAFKA_MSG" ]]; then
    echo "FAIL: no Kafka message"
    tail -30 "$REC_LOG" || true
    exit 1
  fi
  echo "Kafka wire: $(cat "$KAFKA_MSG")"
  python3 - <<PY "$KAFKA_MSG" "$REQ_UUID" "$OUT"
import json, sys
msg_path, req_uuid, resp_json = sys.argv[1], sys.argv[2], sys.argv[3]
parts = open(msg_path).read().strip().split("|")
assert len(parts) == 28, len(parts)
assert parts[2] == "10001" and parts[3] == "cn_ol_item"
assert parts[19] == req_uuid and parts[20] == "900001"
assert "910001" in parts[24] and "LiveRedirect" in parts[24]
assert 910001 in json.loads(resp_json).get("item_ids", [])
print("KAFKA PHASE OK")
PY
  e2e_stop_services
  e2e_free_ports
  unset RECSYS_E2E_KAFKA
fi

# --- Phase 3: rank client LB (duplicate endpoints) ---
if [[ ",${PHASES}," == *",lb,"* ]]; then
  echo ""
  echo "========== PHASE lb =========="
  go test ./pkg/upstream/... -count=1
  export RECSYS_E2E_RANK_ENDPOINTS="http://127.0.0.1:18081,http://127.0.0.1:18081,http://127.0.0.1:18081"
  RECSYS_E2E_REC_LOG="${TMPDIR:-/tmp}/recsys_lb_rec.log"
  e2e_start_services
  e2e_wait_health

  count_rank_hits() {
    if [[ -f "${RANK_LOG:-}" ]]; then
      local c
      c="$(grep -c 'POST /v1/rank/multi' "$RANK_LOG" 2>/dev/null || true)"
      echo "${c:-0}"
    else
      echo 0
    fi
  }
  RANK_BEFORE="$(count_rank_hits | tail -1 | tr -d '[:space:]')"

  for i in 1 2 3 4 5 6; do
    curl -sf -m 15 -X POST http://127.0.0.1:18080/v1/recommend \
      -H 'Content-Type: application/json' \
      -d "{\"uuid\":\"lb-e2e-$i\",\"user_id\":900001,\"exp_ids\":[0],\"ret_count\":5}" \
      | python3 -c 'import json,sys; r=json.load(sys.stdin); assert r.get("item_ids"), r'
  done
  RANK_AFTER="$(count_rank_hits | tail -1 | tr -d '[:space:]')"
  DELTA=$((RANK_AFTER - RANK_BEFORE))
  echo "rank calls delta: $DELTA (expect >= 6)"
  [[ "$DELTA" -ge 6 ]] || { tail -30 "$RANK_LOG"; exit 1; }
  echo "LB PHASE OK"
  e2e_stop_services
fi

echo ""
echo "FULL E2E OK (phases: ${PHASES})"
