#!/usr/bin/env bash
# Full chain E2E: seed Redis (2 users, 10 FM items) -> rank (FM pipeline) -> recommend (recall/merge/filter/rank/show).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
export RECSYS_SEED_REDIS=1
export RECSYS_REDIS_HOST="${RECSYS_REDIS_HOST:-172.31.0.80}"
export RECSYS_REDIS_PORT="${RECSYS_REDIS_PORT:-6379}"
export RECSYS_REDIS_CRYPTO=1
export RECSYS_REDIS_PASSWORD_HEX="${RECSYS_REDIS_PASSWORD_HEX:-d1c98bea6a9824201ac9375488748b3c07}"

echo "==> Seed Redis at ${RECSYS_REDIS_HOST}:${RECSYS_REDIS_PORT}"
python3 scripts/seed_feature_redis.py

for p in 18080 18081; do
  fuser -k "${p}/tcp" >/dev/null 2>&1 || true
done
sleep 1

echo "==> Build services"
go build -o bin/rank-api ./services/rank
go build -o bin/recommend-api ./services/recommend

RANK_LOG="${TMPDIR:-/tmp}/recsys_full_rank.log"
REC_LOG="${TMPDIR:-/tmp}/recsys_full_rec.log"
./bin/rank-api -f services/rank/etc/rank-api.yaml >>"$RANK_LOG" 2>&1 &
RANK_PID=$!
./bin/recommend-api -f services/recommend/etc/recommend-api.yaml >>"$REC_LOG" 2>&1 &
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
READY="$(curl -sf -m 2 http://127.0.0.1:18080/v1/ready)"
echo "ready: $READY"
echo "$READY" | grep -q '"center":true' || { echo "FAIL: center config not loaded"; cat "$REC_LOG"; exit 1; }

echo "==> POST /v1/recommend (user 900001, center pipeline)"
OUT="$(curl -sf -m 15 -X POST http://127.0.0.1:18080/v1/recommend \
  -H 'Content-Type: application/json' \
  -d '{"uuid":"full-e2e","user_id":900001,"exp_ids":[0],"ret_count":5}')"
echo "$OUT"

python3 - <<'PY' "$OUT"
import json, sys
resp = json.loads(sys.argv[1])
ids = resp.get("item_ids") or []
print("item_ids:", ids)
assert 910005 not in ids, "910005 must be filtered by LiveExposure (exposure=15)"
assert 910009 not in ids, "910009 must be filtered by FeatureLess"
assert 910001 in ids, "910001 LiveRedirect should survive merge/filter"
assert len(ids) >= 3, "expected at least 3 items"
# ForcedInsert: LiveRedirect 910001 should be near front
assert ids[0] == 910001, f"ForcedInsert expects 910001 first, got {ids[0]}"
# FM order: highest ctr item 910010 should appear before low ctr among returned
if 910010 in ids and 910004 in ids:
    assert ids.index(910010) < ids.index(910004), "FM rank: 910010 before 910004"
print("FULL CHAIN E2E OK")
PY

echo "==> Logs (tail)"
tail -n 5 "$RANK_LOG" || true
tail -n 5 "$REC_LOG" || true
