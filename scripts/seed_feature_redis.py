#!/usr/bin/env python3
"""
Seed recsys_go E2E data: 2 users + 10 items (910001-910010) for FM 5-feature pipeline.

Keys: recsysgo:user:%d, recsysgo:item:%d (STRING JSON).

Usage:
  RECSYS_SEED_REDIS=1 python3 scripts/seed_feature_redis.py

Env (defaults match recommend/rank yaml):
  RECSYS_REDIS_HOST=172.31.0.80
  RECSYS_REDIS_PORT=6379
  RECSYS_REDIS_PASSWORD_HEX=d1c98bea6a9824201ac9375488748b3c07  # plain: test123
  RECSYS_REDIS_CRYPTO=1
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

    # User 900001: flat semantic + exposure for LiveExposure filter (910005 filtered when limit=3)
    r.set(
        "recsysgo:user:900001",
        json.dumps(
            {
                "age": 38.0,
                "gender": 1.0,
                "income_wan": 6.5,
                "exposure": {"910005": 15},
            },
            separators=(",", ":"),
        ),
    )
    print("SET recsysgo:user:900001")

    # User 900002: nested segments (rank merge path)
    r.set(
        "recsysgo:user:900002",
        json.dumps(
            {
                "user_profile": {"age": 62.0, "gender": 0.0},
                "user_finance": {"income_wan": 8.2},
            },
            separators=(",", ":"),
        ),
    )
    print("SET recsysgo:user:900002")

    # Items 910001-910010: ctr/revenue rise with id -> FM PreRank orders 910010 highest
    for idx in range(10):
        item_id = 910001 + idx
        ctr = 0.012 + 0.014 * idx
        rev = 8000.0 + 7200.0 * idx + (item_id % 97) * 13.0
        doc = {"ctr_7d": round(ctr, 6), "revenue_7d": round(rev, 2)}
        if item_id == 910009:
            doc["feature_less"] = "1"
        key = f"recsysgo:item:{item_id}"
        r.set(key, json.dumps(doc, separators=(",", ":")))
        print("SET", key)

    print("Done: 2 users (900001/900002), 10 items (910001-910010).")
    print("  Filter E2E: user 900001 exposure>3 on 910005; item 910009 feature_less.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
