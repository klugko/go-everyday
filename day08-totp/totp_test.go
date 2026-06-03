package main

import "testing"

// Vecteurs de test de la RFC 6238, annexe B (codes à 8 chiffres, période
// de 30 s). Les secrets sont des ASCII répétés, de longueur adaptée à
// chaque algorithme.
func TestRFC6238Vectors(t *testing.T) {
	var (
		sha1Secret   = []byte("12345678901234567890")
		sha256Secret = []byte("12345678901234567890123456789012")
		sha512Secret = []byte("1234567890123456789012345678901234567890123456789012345678901234")
	)
	cases := []struct {
		unix   int64
		algo   Algo
		secret []byte
		want   string
	}{
		{59, SHA1, sha1Secret, "94287082"},
		{59, SHA256, sha256Secret, "46119246"},
		{59, SHA512, sha512Secret, "90693936"},
		{1111111109, SHA1, sha1Secret, "07081804"},
		{1111111111, SHA1, sha1Secret, "14050471"},
		{1234567890, SHA1, sha1Secret, "89005924"},
		{2000000000, SHA1, sha1Secret, "69279037"},
		{20000000000, SHA1, sha1Secret, "65353130"},
		{1234567890, SHA256, sha256Secret, "91819424"},
		{1234567890, SHA512, sha512Secret, "93441116"},
	}
	for _, c := range cases {
		got := TOTP(c.secret, c.unix, 30, 8, c.algo)
		if got != c.want {
			t.Errorf("TOTP(t=%d, %s) = %s, attendu %s", c.unix, c.algo, got, c.want)
		}
	}
}

func TestDecodeSecret(t *testing.T) {
	// "Hello!" en base32 ; on tolère minuscules, espaces et padding.
	cases := []string{
		"JBSWY3DPEHPK3PXP",
		"jbswy3dpehpk3pxp",
		"JBSW Y3DP EHPK 3PXP",
		"JBSWY3DPEHPK3PXP",
	}
	want, err := DecodeSecret(cases[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases[1:] {
		got, err := DecodeSecret(c)
		if err != nil {
			t.Fatalf("DecodeSecret(%q) : %v", c, err)
		}
		if string(got) != string(want) {
			t.Errorf("DecodeSecret(%q) ne correspond pas à la référence", c)
		}
	}
}

func TestDecodeSecretInvalid(t *testing.T) {
	if _, err := DecodeSecret("not base 32 !!!"); err == nil {
		t.Error("attendu une erreur sur un secret invalide")
	}
}

func TestParseURI(t *testing.T) {
	uri := "otpauth://totp/ACME%20Co:alice@acme.com?secret=JBSWY3DPEHPK3PXP&issuer=ACME%20Co&algorithm=SHA256&digits=8&period=60"
	cfg, err := ParseURI(uri)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Digits != 8 {
		t.Errorf("digits = %d, attendu 8", cfg.Digits)
	}
	if cfg.Period != 60 {
		t.Errorf("period = %d, attendu 60", cfg.Period)
	}
	if cfg.Algo != SHA256 {
		t.Errorf("algo = %s, attendu SHA256", cfg.Algo)
	}
	if cfg.Label != "ACME Co:alice@acme.com" {
		t.Errorf("label = %q", cfg.Label)
	}
}

func TestParseURIDefaults(t *testing.T) {
	// Sans paramètres optionnels : 6 chiffres, 30 s, SHA1.
	cfg, err := ParseURI("otpauth://totp/acme?secret=JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Digits != 6 || cfg.Period != 30 || cfg.Algo != SHA1 {
		t.Errorf("défauts incorrects : %d chiffres, %ds, %s", cfg.Digits, cfg.Period, cfg.Algo)
	}
}

func TestParseURIRejectsHOTP(t *testing.T) {
	if _, err := ParseURI("otpauth://hotp/acme?secret=JBSWY3DPEHPK3PXP&counter=0"); err == nil {
		t.Error("attendu un refus du type hotp")
	}
}
