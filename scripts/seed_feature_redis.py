#!/usr/bin/env python3
"""
Seed recsys_go E2E: profile per entity + merged strategy/recall JSON keys.

Profile: recsysgo:feat:user:%d, recsysgo:feat:item:%d
Filter:  recsysgo:filter:exposure, featureless, label (single key each)
Recall:  recsysgo:recall:lane:{Type}, recsysgo:recall:cf:user:%d (CF only per-user)

Usage: RECSYS_SEED_REDIS=1 python3 scripts/seed_feature_redis.py
"""
import json
import os
import sys


def _aes_decrypt(data: str) -> str:
    from Crypto.Cipher import AES

    ciphertext = bytes.fromhex(data)
    key = bytes.fromhex("23544452656469732d2d3e3230323123")
    block_size = AES.block_size
    cipher = AES.new(key, AES.MODE_ECB)
    plaintext = bytearray()
    prev_block = bytes(block_size)
    iDataSize = len(ciphertext)
    rem = iDataSize % block_size
    if rem == 0:
        pass
    elif rem == 1:
        if iDataSize < block_size + 1:
            raise ValueError("Invalid ciphertext length")
        rem = ciphertext[-1]
        if rem <= 0 or rem >= block_size:
            raise ValueError("Invalid padding byte")
        ciphertext = ciphertext[:-1]
    else:
        raise ValueError("Ciphertext length must be a multiple of block size or have a valid padding byte")

    if len(ciphertext) % block_size != 0:
        raise ValueError("Ciphertext length must be a multiple of block size")

    for i in range(0, len(ciphertext), block_size):
        block = ciphertext[i : i + block_size]
        d_block = cipher.decrypt(block)
        d_block = bytes(x ^ y for x, y in zip(d_block, prev_block))
        plaintext.extend(d_block)
        prev_block = block
    if rem > 0:
        plaintext = plaintext[: -(block_size - rem)]
    return bytes(plaintext).decode("utf-8")


def _create_redis_client(host, port, crypto, password_hex):
    import redis

    password = password_hex
    if str(crypto) in ("1", "true", "True", "yes"):
        password = _aes_decrypt(password_hex)
    r = redis.StrictRedis(host=host, port=port, password=password, db=0)
    r.ping()
    return r


def _del_obsolete_keys(r):
    for pat in (
        "recsysgo:user:*",
        "recsysgo:item:*",
        "recsysgo:filter:exposure:user:*",
        "recsysgo:filter:featureless:item:*",
        "recsysgo:filter:label:item:*",
    ):
        for k in r.scan_iter(pat, count=200):
            kk = k.decode() if isinstance(k, bytes) else k
            r.delete(k)
            print("DEL obsolete", kk)


def main():
    if os.environ.get("RECSYS_SEED_REDIS") != "1":
        print("Set RECSYS_SEED_REDIS=1 to write keys.", file=sys.stderr)
        return 0

    host = os.environ.get("RECSYS_REDIS_HOST", "172.31.0.80")
    port = int(os.environ.get("RECSYS_REDIS_PORT", "6379"))
    pwd_hex = os.environ.get(
        "RECSYS_REDIS_PASSWORD_HEX",
        "d1c98bea6a9824201ac9375488748b3c07",
    )
    r = _create_redis_client(host, port, os.environ.get("RECSYS_REDIS_CRYPTO", "1"), pwd_hex)
    print(f"Connected redis {host}:{port}")
    _del_obsolete_keys(r)

    import time
    now_ts = int(time.time())
    r.set(
        "recsysgo:feat:user:900001",
        json.dumps({
            "age": 38.0,
            "gender": 1.0,
            "income_wan": 6.5,
            "user_segment": "def_group",
            "live_redirect": {
                "map_list": [
                    {"id": 910001, "ts": now_ts, "weight": 1.0},
                    {"id": 910002, "ts": now_ts - 120, "weight": 0.8},
                ]
            },
        }),
    )
    r.set(
        "recsysgo:feat:user:900002",
        json.dumps({
            "user_profile": {"age": 62.0, "gender": 0.0},
            "user_finance": {"income_wan": 8.2},
            "is_new_user": True,
            "user_segment": "T0_NewUser",
        }),
    )
    # item tag = category id 0..5 (see README); used to build invert index
    item_tags = {
        910001: 1, 910002: 1, 910003: 2, 910004: 2, 910005: 3,
        910006: 3, 910007: 4, 910008: 4, 910009: 5, 910010: 5,
    }
    invert_by_tag = {}
    for idx in range(10):
        item_id = 910001 + idx
        tag = item_tags[item_id]
        ctr = 0.012 + 0.014 * idx
        rev = 8000.0 + 7200.0 * idx + (item_id % 97) * 13.0
        if item_id == 910009:
            # FeatureLess E2E: no recsysgo:feat:item key (cannot rank)
            invert_by_tag.setdefault(tag, []).append(item_id)
            continue
        r.set(
            f"recsysgo:feat:item:{item_id}",
            json.dumps({"tag": tag, "ctr_7d": round(ctr, 6), "revenue_7d": round(rev, 2)}),
        )
        invert_by_tag.setdefault(tag, []).append(item_id)

    for tag, ids in sorted(invert_by_tag.items()):
        key = f"recsysgo:recall:invert:tag:{tag}"
        r.set(key, json.dumps(ids))
        print("SET", key, ids)

    # CrossTag7d: user tag interest 7d (personalized, C++ tag_time_7d)
    r.set(
        "recsysgo:recall:taginterest:7d:user:900001",
        json.dumps([{"tag": 3, "weight": 0.7}, {"tag": 4, "weight": 0.3}]),
    )
    r.set(
        "recsysgo:recall:taginterest:7d:user:900002",
        json.dumps([{"tag": 1, "weight": 0.8}, {"tag": 2, "weight": 0.2}]),
    )
    print("SET recsysgo:recall:taginterest:7d:user:900001/900002")

    r.set("recsysgo:filter:exposure", json.dumps({"910005": 15}))
    # LiveRedirect: per-user live_redirect in feat:user (not lane list); lane key kept for legacy tools only
    r.set("recsysgo:homogen:exchange", json.dumps({"910010": 910003}))
    r.set("recsysgo:recall:lane:HotMap", json.dumps([910002, 910003, 910004, 910010]))
    r.set("recsysgo:recall:lane:NewUser_Hot", json.dumps([910002, 910003, 910004]))
    r.set("recsysgo:recall:lane:NewUser_HighRetention", json.dumps([910001, 910010]))
  # CF list disjoint from CrossTag invert (tag 3/4) so merge keeps CrossTag7d recall_type on 910006-910008
    r.set("recsysgo:recall:cf:user:900001", json.dumps([910010, 910004, 910003, 910002]))
    r.set("recsysgo:recall:cf:user:900002", json.dumps([910010, 910008, 910007, 910006]))

    print("Done: feat per entity; filter/recall merged keys.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
