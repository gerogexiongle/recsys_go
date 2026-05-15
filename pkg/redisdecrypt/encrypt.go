package redisdecrypt

import (
	"crypto/aes"
	"encoding/hex"
)

// EncryptPassword encrypts plaintext for Redis config (matches C++ Crypto::cbc_encrypt + AES).
func EncryptPassword(plain string, key []byte) (string, error) {
	if key == nil {
		key = defaultKey
	}
	blockSize := aes.BlockSize
	data := []byte(plain)
	rem := len(data) % blockSize

	cipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	buf := make([]byte, blockSize)
	var out []byte
	for off := 0; off < len(data); off += blockSize {
		n := blockSize
		if off+n > len(data) {
			n = len(data) - off
		}
		for i := 0; i < n; i++ {
			buf[i] ^= data[off+i]
		}
		block := make([]byte, blockSize)
		cipher.Encrypt(block, buf)
		copy(buf, block)
		out = append(out, block...)
	}
	if rem != 0 {
		out = append(out, byte(rem))
	}
	return hex.EncodeToString(out), nil
}
