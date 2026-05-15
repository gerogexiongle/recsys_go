#!/usr/bin/env python3
"""
Seed recsys_go E2E data: profile feat keys + separate filter strategy keys.

Profile (FM / rank):
  recsysgo:feat:user:%d
  recsysgo:feat:item:%d

Filter strategies (center only; missing key => strategy inactive, see pkg/featurestore/strategy.go):
  recsysgo:filter:exposure:user:%d   JSON {"910005":15}
  recsysgo:filter:featureless:item:%d  "1" only when item should be filtered
  recsysgo:filter:label:item:%d      plain label string (optional)

Usage:
  RECSYS_SEED_REDIS=1 python3 scripts/seed_feature_redis.py
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
    redis_client = redis.StrictRedis(host=host, port=port, password=password, db=0)
    redis_client.ping()
    return redis_client


def _del_legacy_keys(r):
    legacy = [f"recsysgo:user:{u}" for u in (900001, 900002)]
    legacy += [f"recsysgo:item:{i}" for i in range(910001, 910011)]
    for k in legacy:
        if r.delete(k):
            print("DEL legacy", k)


def main():
    if os.environ.get("RECSYS_SEED_REDIS") != "1":
        print("Set RECSYS_SEED_REDIS=1 to write keys.", file=sys.stderr)
        return 0

    host = os.environ.get("RECSYS_REDIS_HOST", "172.31.0.80")
    port = int(os.environ.get("RECSYS_REDIS_PORT", "6379"))
    crypto = os.environ.get("RECSYS_REDIS_CRYPTO", "1")
    pwd_hex = os.environ.get(
        "RECSYS_REDIS_PASSWORD_HEX",
        "d1c98bea6a9824201ac9375488748b3c07",
    )

    r = _create_redis_client(host, port, crypto, pwd_hex)
    print(f"Connected redis {host}:{port}")
    _del_legacy_keys(r)

    # Profile: user 900001 (flat semantic)
    r.set(
        "recsysgo:feat:user:900001",
        json.dumps({"age": 38.0, "gender": 1.0, "income_wan": 6.5}, separators=(",", ":")),
    )
    print("SET recsysgo:feat:user:900001")

    # Profile: user 900002 (nested segments for rank merge path)
    r.set(
        "recsysgo:feat:user:900002",
        json.dumps(
            {
                "user_profile": {"age": 62.0, "gender": 0.0},
                "user_finance": {"income_wan": 8.2},
            },
            separators=(",", ":"),
        ),
    )
    print("SET recsysgo:feat:user:900002")

    # Strategy: LiveExposure (separate from profile; C++ game_exposure field)
    r.set(
        "recsysgo:filter:exposure:user:900001",
        json.dumps({"910005": 15}, separators=(",", ":")),
    )
    print("SET recsysgo:filter:exposure:user:900001")

    for idx in range(10):
        item_id = 910001 + idx
        ctr = 0.012 + 0.014 * idx
        rev = 8000.0 + 7200.0 * idx + (item_id % 97) * 13.0
        feat_key = f"recsysgo:feat:item:{item_id}"
        r.set(
            feat_key,
            json.dumps({"ctr_7d": round(ctr, 6), "revenue_7d": round(rev, 2)}, separators=(",", ":")),
        )
        print("SET", feat_key)

        if item_id == 910009:
            fl_key = f"recsysgo:filter:featureless:item:{item_id}"
            r.set(fl_key, "1")
            print("SET", fl_key)
        # 910001-910008, 910010: no featureless key => FeatureLess keeps item

    print("Done: 2 users, 10 items (profile + strategy keys).")
    print("  LiveExposure: filter:exposure:user:900001 -> 910005 filtered")
    print("  FeatureLess: filter:featureless:item:910009 only")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
