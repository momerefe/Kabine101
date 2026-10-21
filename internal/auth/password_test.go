package auth

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestPasswordHashIsSaltedAndVerifiesOnlyOriginal(t *testing.T) {
	password := "A long private password 314159"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !strings.HasPrefix(first, "$argon2id$v=19$m=65536,t=3,p=4$") {
		t.Fatal("hash is not independently salted Argon2id")
	}
	if !checkPassword(first, password) || checkPassword(first, "Another long private password") {
		t.Fatal("password comparison failed")
	}
	for _, bad := range []string{"plaintext", strings.Replace(first, "m=65536", "m=4294967295", 1), strings.Replace(first, "v=19", "v=18", 1), first + "$extra"} {
		if checkPassword(bad, password) {
			t.Fatal("malformed or unbounded parameters accepted")
		}
	}
}

func TestPasswordPolicyUsesCharactersAndBoundedInput(t *testing.T) {
	for _, p := range []string{strings.Repeat("x", 14), strings.Repeat("ş", 129), "password contains\x00a nul", string([]byte{0xff})} {
		if validPassword(p) {
			t.Fatal("invalid password accepted")
		}
	}
	for _, p := range []string{strings.Repeat("ş", 15), strings.Repeat("x", 128)} {
		if !validPassword(p) {
			t.Fatal("valid password rejected")
		}
	}
}

func TestOperatorPasswordUsesSameHashWithSeparateMinimum(t *testing.T) {
	password := "246813579"
	if _, err := HashPassword(password); err == nil {
		t.Fatal("short operator password accepted by normal account policy")
	}
	encoded, err := HashOperatorPassword(password)
	if err != nil || !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=4$") || !checkPassword(encoded, password) {
		t.Fatal("operator password did not use the standard Argon2id hash")
	}
	for _, password := range []string{strings.Repeat("x", 7), strings.Repeat("x", 129), "abcdefgh\x00", string([]byte{0xff})} {
		if _, err := HashOperatorPassword(password); err == nil {
			t.Fatal("invalid operator password accepted")
		}
	}
	for _, password := range []string{strings.Repeat("ş", 8), strings.Repeat("x", 128)} {
		if !validPasswordLength(password, 8) {
			t.Fatal("valid operator password rejected")
		}
	}
}

func TestTokensHave256BitsAndOnlyHashCanBePersisted(t *testing.T) {
	first, hash, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil || len(raw) != 32 || len(hash) != 32 || first == second || bytes.Equal(raw, hash) {
		t.Fatal("token entropy or hashing failed")
	}
	got, ok := tokenHash(first)
	if !ok || !bytes.Equal(got, hash) {
		t.Fatal("token did not round-trip")
	}
	for _, bad := range []string{"", first + "=", first[:42], strings.Repeat("!", 43)} {
		if _, ok := tokenHash(bad); ok {
			t.Fatal("non-canonical token accepted")
		}
	}
}
