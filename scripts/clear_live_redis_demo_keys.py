#!/usr/bin/env python3
"""Remove recsys_go demo keys from live Redis. Usage: RECSYS_CLEAR_LIVE_REDIS=1 python3 scripts/clear_live_redis_demo_keys.py"""
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
    if rem == 1:
        ciphertext = ciphertext[:-1]
    for i in range(0, len(ciphertext), block_size):
        block = ciphertext[i : i + block_size]
        d_block = cipher.decrypt(block)
        d_block = bytes(x ^ y for x, y in zip(d_block, prev_block))
        plaintext.extend(d_block)
        prev_block = block
    return bytes(plaintext).decode("utf-8")


def main():
    if os.environ.get("RECSYS_CLEAR_LIVE_REDIS") != "1":
        print("Set RECSYS_CLEAR_LIVE_REDIS=1", file=sys.stderr)
        return 0
    import redis

    r = redis.StrictRedis(
        host="algo-cn-live-redis.mini1.cn",
        port=6379,
        password=_aes_decrypt("78144d064ed8cd728be1b5ebb7fdb1e8"),
    )
    r.ping()
    keys = [
        "recsysgo:filter:exposure",
        "recsysgo:filter:featureless",
        "recsysgo:filter:label",
        "recsysgo:recall:lane:LiveRedirect",
    ]
    for u in (900001, 900002):
        keys.append(f"recsysgo:feat:user:{u}")
        keys.append(f"recsysgo:recall:cf:user:{u}")
    for i in range(910001, 910011):
        keys.append(f"recsysgo:feat:item:{i}")
    for pat in ("recsysgo:user:*", "recsysgo:item:*", "recsysgo:filter:*:user:*", "recsysgo:filter:*:item:*"):
        for k in r.scan_iter(pat, count=500):
            keys.append(k.decode() if isinstance(k, bytes) else k)
    seen = set()
    for k in keys:
        if k in seen:
            continue
        seen.add(k)
        print("DEL", k, "ok" if r.delete(k) else "absent")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
