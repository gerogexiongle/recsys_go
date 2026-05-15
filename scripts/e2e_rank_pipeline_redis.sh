#!/usr/bin/env bash
# Pipeline FM + Redis JSON (CN test Redis). Requires network + RECSYS_REDIS_PASSWORD_HEX (encrypted hex).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export RECSYS_REDIS_PASSWORD_HEX="${RECSYS_REDIS_PASSWORD_HEX:-b5e4cd9f0953b5b1ba6bcb8a0354a4980a}"

RECSYS_SEED_REDIS=1 python3 scripts/seed_feature_redis.py

fuser -k 18081/tcp >/dev/null 2>&1 || true
sleep 0.5

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
go build -o bin/rank-api ./services/rank

./bin/rank-api -f services/rank/etc/rank-api.pipeline-e2e.yaml &
RANK_PID=$!
trap 'kill $RANK_PID 2>/dev/null || true; wait $RANK_PID 2>/dev/null || true' EXIT

for _ in $(seq 1 40); do
  if curl -sf -m 1 http://127.0.0.1:18081/health >/dev/null; then
    break
  fi
  sleep 0.25
done

curl -sf -m 2 http://127.0.0.1:18081/health | grep -qx ok

OUT="$(curl -sf -m 8 -X POST http://127.0.0.1:18081/v1/rank/multi \
  -H 'Content-Type: application/json' \
  -d '{"uuid":"e2e-redis","user_id":900001,"item_groups":[{"name":"g","item_ids":[910001,910002,910003,910004,910005,910006,910007,910008,910009,910010],"ret_count":3}]}')"
echo "$OUT"
echo "$OUT" | python3 -c "import json,sys; j=json.load(sys.stdin); ids=[x['item_id'] for x in j['ranked_groups'][0]['item_scores']]; want=[910010,910009,910008]; assert ids==want, ids" || { echo "E2E FAIL: want top3 ctr order [910010,910009,910008]"; exit 1; }
echo "E2E rank+redis OK"
