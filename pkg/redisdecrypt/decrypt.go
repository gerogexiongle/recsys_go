// Package redisdecrypt implements the same password decryption used by Python Redis helpers (ECB + CBC-style XOR chain).
package redisdecrypt

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"
)

var defaultKey, _ = hex.DecodeString("23544452656469732d2d3e3230323123")

// DecryptPassword decrypts hex-encoded ciphertext used for Redis passwords (matches legacy Python helper).
func DecryptPassword(cipherHex string, key []byte) (string, error) {
	if key == nil {
		key = defaultKey
	}
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", err
	}
	blockSize := aes.BlockSize
	if len(ciphertext) < blockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	origRem := len(ciphertext) % blockSize
	trimPad := 0
	if origRem == 0 {
	} else if origRem == 1 {
		if len(ciphertext) < blockSize+1 {
			return "", fmt.Errorf("invalid ciphertext length")
		}
		pad := int(ciphertext[len(ciphertext)-1])
		if pad <= 0 || pad >= blockSize {
			return "", fmt.Errorf("invalid padding byte")
		}
		trimPad = pad
		ciphertext = ciphertext[:len(ciphertext)-1]
		if len(ciphertext)%blockSize != 0 {
			return "", fmt.Errorf("ciphertext length after strip")
		}
	} else {
		return "", fmt.Errorf("ciphertext length mod block must be 0 or 1")
	}

	cipher, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plain := make([]byte, 0, len(ciphertext))
	prev := make([]byte, blockSize)
	for i := 0; i < len(ciphertext); i += blockSize {
		block := ciphertext[i : i+blockSize]
		buf := make([]byte, blockSize)
		cipher.Decrypt(buf, block)
		for j := 0; j < blockSize; j++ {
			buf[j] ^= prev[j]
		}
		plain = append(plain, buf...)
		copy(prev, block)
	}

	if trimPad > 0 {
		if len(plain) < blockSize-trimPad {
			return "", fmt.Errorf("invalid trim")
		}
		plain = plain[:len(plain)-(blockSize-trimPad)]
	}
	return string(plain), nil
}
