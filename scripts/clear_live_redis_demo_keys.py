#!/usr/bin/env python3
"""
Remove recsys_go demo keys mistakenly written to algo-cn-live-redis.mini1.cn.

Requires: pip install redis pycryptodome

Usage:
  RECSYS_CLEAR_LIVE_REDIS=1 python3 scripts/clear_live_redis_demo_keys.py
"""
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


def main():
    if os.environ.get("RECSYS_CLEAR_LIVE_REDIS") != "1":
        print("Set RECSYS_CLEAR_LIVE_REDIS=1 to delete keys on live Redis.", file=sys.stderr)
        return 0

    import redis

    password = _aes_decrypt("78144d064ed8cd728be1b5ebb7fdb1e8")
    r = redis.StrictRedis(host="algo-cn-live-redis.mini1.cn", port=6379, password=password)
    r.ping()

    keys = [f"recsysgo:user:{u}" for u in (900001, 900002)]
    keys += [f"recsysgo:item:{i}" for i in range(910001, 910011)]

    for k in keys:
        n = r.delete(k)
        print("DEL", k, "ok" if n else "absent")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
