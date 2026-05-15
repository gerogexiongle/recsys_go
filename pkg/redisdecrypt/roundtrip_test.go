package redisdecrypt

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	for _, plain := range []string{"test123", "b5e4cd9f0953b5b1ba6bcb8a0354a4980a", ""} {
		if plain == "" {
			continue
		}
		hexCipher, err := EncryptPassword(plain, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecryptPassword(hexCipher, nil)
		if err != nil {
			t.Fatalf("decrypt %q: %v", hexCipher, err)
		}
		if got != plain {
			t.Fatalf("plain %q -> %s -> %q", plain, hexCipher, got)
		}
	}
}

func TestEncryptTest123Hex(t *testing.T) {
	hexCipher, err := EncryptPassword("test123", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("test123 PasswordHex:", hexCipher)
}
