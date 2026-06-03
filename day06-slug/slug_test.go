package main

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Le cas qui donne son nom à l'exercice.
		{"Café à Tana", "cafe-a-tana"},
		// Accents dans tous les sens.
		{"Élève à l'œuvre", "eleve-a-l-oeuvre"},
		{"Crème brûlée & Cœur", "creme-brulee-coeur"},
		// Ponctuation et espaces multiples se réduisent à un séparateur.
		{"  Hello,   World!!  ", "hello-world"},
		{"déjà-vu", "deja-vu"},
		// Chiffres conservés, symboles abandonnés.
		{"Go 1.26 — c'est parti", "go-1-26-c-est-parti"},
		// Allemand : ß -> ss, ü -> u.
		{"Straße über München", "strasse-uber-munchen"},
		// Rien d'exploitable -> chaîne vide, pas de séparateur orphelin.
		{"!!! ??? ...", ""},
		{"", ""},
		// Les emojis et autres runes inconnues sont des frontières.
		{"pizza 🍕 party", "pizza-party"},
	}
	for _, c := range cases {
		if got := Slugify(c.in, Options{Lower: true}); got != c.want {
			t.Errorf("Slugify(%q) = %q, attendu %q", c.in, got, c.want)
		}
	}
}

func TestSlugifyKeepCase(t *testing.T) {
	// Sans minusculisation, É reste E majuscule après translittération.
	got := Slugify("École Élémentaire", Options{Lower: false})
	want := "Ecole-Elementaire"
	if got != want {
		t.Errorf("Slugify keep-case = %q, attendu %q", got, want)
	}
}

func TestSlugifyCustomSep(t *testing.T) {
	got := Slugify("Café à Tana", Options{Lower: true, Sep: "_"})
	want := "cafe_a_tana"
	if got != want {
		t.Errorf("Slugify sep=_ = %q, attendu %q", got, want)
	}
}

func TestSlugifyMax(t *testing.T) {
	// Coupe à 8 octets max sans laisser un mot tronqué : "the-quick"
	// ferait 9, donc on rogne au séparateur précédent -> "the".
	got := Slugify("the quick brown fox", Options{Lower: true, Max: 8})
	want := "the"
	if got != want {
		t.Errorf("Slugify max=8 = %q, attendu %q", got, want)
	}
	// Sans coupure si on tient dans la limite.
	if got := Slugify("the quick", Options{Lower: true, Max: 9}); got != "the-quick" {
		t.Errorf("Slugify max=9 = %q, attendu the-quick", got)
	}
}
