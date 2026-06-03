package main

import (
	"math"
	"strings"
	"testing"
)

func TestGeneratePasswordRespectsLength(t *testing.T) {
	p := Policy{Length: 24, Lower: true, Upper: true, Digits: true, Symbols: true}
	pw, err := GeneratePassword(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(pw) != 24 {
		t.Errorf("longueur = %d, attendu 24", len(pw))
	}
}

func TestGeneratePasswordGuaranteesClasses(t *testing.T) {
	p := Policy{Length: 8, Lower: true, Upper: true, Digits: true, Symbols: true}
	// On tire plusieurs fois : chaque tirage doit contenir au moins un
	// caractère de chaque classe demandée.
	for n := 0; n < 200; n++ {
		pw, err := GeneratePassword(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.ContainsAny(pw, lowers) ||
			!strings.ContainsAny(pw, uppers) ||
			!strings.ContainsAny(pw, digits) ||
			!strings.ContainsAny(pw, symbols) {
			t.Fatalf("tirage %q sans toutes les classes", pw)
		}
	}
}

func TestNoAmbiguous(t *testing.T) {
	p := Policy{Length: 64, Lower: true, Upper: true, Digits: true, NoAmbiguous: true}
	for n := 0; n < 50; n++ {
		pw, err := GeneratePassword(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(pw, ambiguous) {
			t.Fatalf("caractère ambigu dans %q", pw)
		}
	}
}

func TestPasswordTooShortForClasses(t *testing.T) {
	// 3 caractères ne peuvent pas porter 4 classes garanties.
	p := Policy{Length: 3, Lower: true, Upper: true, Digits: true, Symbols: true}
	if _, err := GeneratePassword(p); err == nil {
		t.Error("attendu une erreur quand la longueur < nombre de classes")
	}
}

func TestNoClassesIsError(t *testing.T) {
	if _, err := GeneratePassword(Policy{Length: 10}); err == nil {
		t.Error("attendu une erreur quand aucune classe n'est activée")
	}
}

func TestPassphraseStructure(t *testing.T) {
	pp, err := GeneratePassphrase(5, "-")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(pp, "-") + 1; got != 5 {
		t.Errorf("%q contient %d mots, attendu 5", pp, got)
	}
}

func TestWordlistIsFull(t *testing.T) {
	// La liste EFF « large » fait exactement 7776 mots (6^5).
	if len(words) != 7776 {
		t.Errorf("liste de %d mots, attendu 7776", len(words))
	}
}

func TestPassphraseEntropy(t *testing.T) {
	// 6 mots sur 7776 ~ 77,5 bits (12,92 bits par mot).
	got := PassphraseEntropy(6)
	if math.Abs(got-77.5) > 0.5 {
		t.Errorf("entropie 6 mots = %.2f, attendu ~77,5", got)
	}
}

func TestPasswordEntropy(t *testing.T) {
	// 20 caractères, pool de 26 minuscules : 20 * log2(26) ~ 94 bits.
	got := PasswordEntropy(Policy{Length: 20, Lower: true})
	if math.Abs(got-94.0) > 0.5 {
		t.Errorf("entropie = %.2f, attendu ~94", got)
	}
}
