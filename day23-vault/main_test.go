package main

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundTrip(t *testing.T) {
	in := map[string]string{
		"github": "ghp_secret_token",
		"db":     "postgres://user:pass@host/db",
	}
	blob, err := seal(in, "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	out, err := open(blob, "hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("attendu %d secrets, obtenu %d", len(in), len(out))
	}
	for k, v := range in {
		if out[k] != v {
			t.Errorf("%q : attendu %q, obtenu %q", k, v, out[k])
		}
	}
}

func TestWrongPassword(t *testing.T) {
	blob, err := seal(map[string]string{"a": "b"}, "bon-mot-de-passe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := open(blob, "mauvais"); err != errBadPassword {
		t.Fatalf("attendu errBadPassword, obtenu %v", err)
	}
}

// Un octet modifié dans le chiffré doit faire échouer l'ouverture : c'est tout
// l'intérêt du tag d'authentification de GCM.
func TestTamperingDetected(t *testing.T) {
	blob, err := seal(map[string]string{"a": "b"}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	blob[len(blob)-1] ^= 0xff
	if _, err := open(blob, "pw"); err != errBadPassword {
		t.Fatalf("altération non détectée : %v", err)
	}
}

// Deux scellements du même contenu doivent différer : sel et nonce aléatoires
// interdisent de deviner que deux coffres contiennent la même chose.
func TestSaltMakesOutputUnique(t *testing.T) {
	a, err := seal(map[string]string{"x": "y"}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	b, err := seal(map[string]string{"x": "y"}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Error("deux scellements identiques : sel/nonce non aléatoires ?")
	}
}

func TestEmptyVault(t *testing.T) {
	blob, err := seal(map[string]string{}, "pw")
	if err != nil {
		t.Fatal(err)
	}
	out, err := open(blob, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Errorf("coffre vide attendu, obtenu %v", out)
	}
}
