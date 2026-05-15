#!/usr/bin/env bash
# E2E: recommend -> rank with duplicate Endpoints (same host 3x) to exercise client round_robin.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export RECSYS_SEED_REDIS=1
export RECSYS_REDIS_HOST="${RECSYS_REDIS_HOST:-172.31.0.80}"
export RECSYS_REDIS_PORT="${RECSYS_REDIS_PORT:-6379}"
export RECSYS_REDIS_CRYPTO=1
export RECSYS_REDIS_PASSWORD_HEX="${RECSYS_REDIS_PASSWORD_HEX:-d1c98bea6a9824201ac9375488748b3c07}"

echo "==> unit: upstream LB (duplicate endpoints)"
go test ./pkg/upstream/... -count=1

echo "==> Seed Redis"
python3 scripts/seed_feature_redis.py

for p in 18080 18081; do
  fuser -k "${p}/tcp" >/dev/null 2>&1 || true
done
sleep 1

go build -o bin/rank-api ./services/rank
go build -o bin/recommend-api ./services/recommend

LB_YAML="services/recommend/etc/recommend-api-lb-test.yaml"

RANK_LOG="${TMPDIR:-/tmp}/recsys_lb_rank.log"
REC_LOG="${TMPDIR:-/tmp}/recsys_lb_rec.log"
./bin/rank-api -f services/rank/etc/rank-api.yaml >>"$RANK_LOG" 2>&1 &
RANK_PID=$!
./bin/recommend-api -f "$LB_YAML" >>"$REC_LOG" 2>&1 &
REC_PID=$!
trap 'kill $REC_PID $RANK_PID 2>/dev/null || true; wait $REC_PID $RANK_PID 2>/dev/null || true' EXIT

echo "==> Wait for health"
for _ in $(seq 1 40); do
  if curl -sf -m 1 http://127.0.0.1:18081/health >/dev/null && \
     curl -sf -m 1 http://127.0.0.1:18080/health >/dev/null; then
    break
  fi
  sleep 0.25
done
curl -sf -m 2 http://127.0.0.1:18081/health | grep -qx ok
curl -sf -m 2 http://127.0.0.1:18080/health | grep -qx ok

count_rank_hits() {
  if [ -f "$1" ]; then
    grep -c 'POST /v1/rank/multi' "$1" 2>/dev/null || true
  else
    echo 0
  fi
}
RANK_BEFORE="$(count_rank_hits "$RANK_LOG")"
RANK_BEFORE="${RANK_BEFORE:-0}"

echo "==> 6x POST /v1/recommend (each triggers rank via 3-endpoint RR)"
for i in 1 2 3 4 5 6; do
  OUT="$(curl -sf -m 15 -X POST http://127.0.0.1:18080/v1/recommend \
    -H 'Content-Type: application/json' \
    -d "{\"uuid\":\"lb-e2e-$i\",\"user_id\":900001,\"exp_ids\":[0],\"ret_count\":5}")"
  echo "$OUT" | python3 -c 'import json,sys; r=json.load(sys.stdin); assert r.get("item_ids"), r'
done

RANK_AFTER="$(count_rank_hits "$RANK_LOG")"
RANK_AFTER="${RANK_AFTER:-0}"
DELTA=$((RANK_AFTER - RANK_BEFORE))
echo "rank /v1/rank/multi log lines delta: $DELTA (expect >= 6)"
if [ "$DELTA" -lt 6 ]; then
  echo "FAIL: expected at least 6 rank calls"
  tail -n 30 "$RANK_LOG"
  exit 1
fi

python3 - <<'PY'
import json, urllib.request
out = urllib.request.urlopen(
    urllib.request.Request(
        "http://127.0.0.1:18080/v1/recommend",
        data=b'{"uuid":"lb-check","user_id":900001,"exp_ids":[0],"ret_count":5}',
        headers={"Content-Type": "application/json"},
        method="POST",
    ),
    timeout=15,
).read()
resp = json.loads(out)
assert 910001 in resp.get("item_ids", []), resp
print("LB DUP ENDPOINTS E2E OK")
PY
