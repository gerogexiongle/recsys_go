#!/usr/bin/env bash
# Local E2E: free ports, start rank + recommend, curl health + recommend, shutdown.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for p in 18080 18081; do
  fuser -k "${p}/tcp" >/dev/null 2>&1 || true
done
sleep 1

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
go build -o bin/rank-api ./services/rank
go build -o bin/recommend-api ./services/recommend

RANK_LOG="${TMPDIR:-/tmp}/recsys_e2e_rank.log"
REC_LOG="${TMPDIR:-/tmp}/recsys_e2e_rec.log"
./bin/rank-api -f services/rank/etc/rank-api.yaml >>"$RANK_LOG" 2>&1 &
RANK_PID=$!
./bin/recommend-api -f services/recommend/etc/recommend-api.yaml >>"$REC_LOG" 2>&1 &
REC_PID=$!
trap 'kill $REC_PID $RANK_PID 2>/dev/null || true; wait $REC_PID $RANK_PID 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  if curl -sf -m 1 http://127.0.0.1:18081/health >/dev/null && curl -sf -m 1 http://127.0.0.1:18080/health >/dev/null; then
    break
  fi
  sleep 0.3
done

curl -sf -m 2 http://127.0.0.1:18081/health | grep -qx ok
curl -sf -m 2 http://127.0.0.1:18080/health | grep -qx ok

OUT="$(curl -sf -m 5 -X POST http://127.0.0.1:18080/v1/recommend \
  -H 'Content-Type: application/json' \
  -d '{"uuid":"e2e","user_id":1,"ret_count":2}')"
echo "$OUT"
echo "$OUT" | grep -q '\[10003,10002\]' || { echo "E2E FAIL: expected item_ids [10003,10002]"; exit 1; }
echo "E2E OK"
