package main

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	plain := []byte("un secret bien gardé\x00\x01\x02 avec des octets binaires")
	enc, err := encrypt(plain, "motdepasse")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(enc, plain) {
		t.Fatal("le texte clair se retrouve tel quel dans le chiffré")
	}
	dec, err := decrypt(enc, "motdepasse")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("aller-retour cassé : %q != %q", dec, plain)
	}
}

func TestWrongPassword(t *testing.T) {
	enc, err := encrypt([]byte("data"), "bon")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decrypt(enc, "mauvais"); err != errBadPassword {
		t.Fatalf("attendu errBadPassword, obtenu %v", err)
	}
}

func TestNotOurFile(t *testing.T) {
	if _, err := decrypt([]byte("juste du texte au hasard"), "pw"); err != errNotEncrypted {
		t.Fatalf("attendu errNotEncrypted, obtenu %v", err)
	}
}

func TestTamperingDetected(t *testing.T) {
	enc, err := encrypt([]byte("data"), "pw")
	if err != nil {
		t.Fatal(err)
	}
	enc[len(enc)-1] ^= 0xff
	if _, err := decrypt(enc, "pw"); err != errBadPassword {
		t.Fatalf("altération non détectée : %v", err)
	}
}

func TestEmptyFile(t *testing.T) {
	enc, err := encrypt(nil, "pw")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := decrypt(enc, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(dec) != 0 {
		t.Fatalf("attendu vide, obtenu %d octets", len(dec))
	}
}

func TestOutName(t *testing.T) {
	cases := []struct {
		explicit, in string
		dec          bool
		want         string
	}{
		{"", "a.pdf", false, "a.pdf.enc"},
		{"", "a.pdf.enc", true, "a.pdf"},
		{"", "a.pdf", true, "a.pdf.dec"},
		{"forced.bin", "a.pdf", false, "forced.bin"},
	}
	for _, c := range cases {
		if got := outName(c.explicit, c.in, c.dec); got != c.want {
			t.Errorf("outName(%q,%q,%v) = %q, attendu %q", c.explicit, c.in, c.dec, got, c.want)
		}
	}
}
