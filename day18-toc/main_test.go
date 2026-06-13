package main

import (
	"strings"
	"testing"
)

func TestUpdateInsertsAfterFirstHeading(t *testing.T) {
	md := "# Titre\n\nun peu de texte\n\n## Partie\n\nencore du texte\n"
	got := update(md)

	if !strings.Contains(got, openTag) || !strings.Contains(got, closeTag) {
		t.Fatalf("les marqueurs devraient être posés :\n%s", got)
	}
	// la table doit lister les deux titres, avec le ## indenté sous le #.
	for _, want := range []string{"- [Titre](#titre)", "  - [Partie](#partie)"} {
		if !strings.Contains(got, want) {
			t.Errorf("table devrait contenir %q :\n%s", want, got)
		}
	}
	// elle se glisse après le titre, pas avant.
	if strings.Index(got, openTag) < strings.Index(got, "# Titre") {
		t.Error("la table devrait venir après le premier titre")
	}
	// le corps d'origine reste là.
	if !strings.Contains(got, "encore du texte") {
		t.Error("le contenu d'origine ne devrait pas disparaître")
	}
}

func TestUpdateReplacesBetweenMarkers(t *testing.T) {
	md := "# Doc\n\n" + openTag + "\nvieille table périmée\n" + closeTag + "\n\n## Neuf\n"
	got := update(md)

	if strings.Contains(got, "vieille table périmée") {
		t.Error("l'ancienne table devrait être remplacée")
	}
	if !strings.Contains(got, "- [Neuf](#neuf)") {
		t.Errorf("la nouvelle table devrait lister Neuf :\n%s", got)
	}
}

func TestUpdateIsIdempotent(t *testing.T) {
	md := "# A\n\n## B\n\ntexte\n"
	once := update(md)
	twice := update(once)
	if once != twice {
		t.Errorf("repasser ne devrait rien changer\n  une fois : %q\n  deux fois: %q", once, twice)
	}
}

func TestUpdateNoHeadingsLeavesUntouched(t *testing.T) {
	md := "juste du texte\nsans aucun titre\n"
	if got := update(md); got != md {
		t.Errorf("sans titre on ne touche à rien, reçu %q", got)
	}
}

func TestUpdatePlaceholderMarker(t *testing.T) {
	// un marqueur d'ouverture seul, posé au milieu : la table s'y insère.
	md := "intro\n\n" + openTag + "\n\n# Vrai titre\n"
	got := update(md)
	if strings.Index(got, "- [Vrai titre]") < strings.Index(got, "intro") {
		t.Errorf("la table devrait remplacer le marqueur sur place :\n%s", got)
	}
	if !strings.Contains(got, closeTag) {
		t.Error("le marqueur de fermeture devrait être ajouté")
	}
}

func TestHeadingsSkipsCodeBlocks(t *testing.T) {
	md := "# Vrai\n\n```sh\n# pas un titre\n```\n\n## Aussi vrai\n"
	heads := headings(strings.Split(md, "\n"))
	if len(heads) != 2 {
		t.Fatalf("2 titres attendus, %d trouvés : %+v", len(heads), heads)
	}
	if heads[0].text != "Vrai" || heads[1].text != "Aussi vrai" {
		t.Errorf("titres inattendus : %+v", heads)
	}
}

func TestParseHeading(t *testing.T) {
	cases := []struct {
		in    string
		level int
		text  string
	}{
		{"# Titre", 1, "Titre"},
		{"### Sous-section", 3, "Sous-section"},
		{"####### trop", 0, ""},
		{"#collé", 0, ""},
		{"## ", 0, ""},
		{"pas un titre", 0, ""},
	}
	for _, c := range cases {
		if level, text := parseHeading(c.in); level != c.level || text != c.text {
			t.Errorf("parseHeading(%q) = (%d, %q), attendu (%d, %q)", c.in, level, text, c.level, c.text)
		}
	}
}

func TestSlugDedup(t *testing.T) {
	seen := map[string]int{}
	want := []struct{ in, out string }{
		{"Mise en place", "mise-en-place"},
		{"C'est parti !", "cest-parti-"},
		{"Mise en place", "mise-en-place-1"},
		{"Mise en place", "mise-en-place-2"},
	}
	for _, c := range want {
		if got := slug(c.in, seen); got != c.out {
			t.Errorf("slug(%q) = %q, attendu %q", c.in, got, c.out)
		}
	}
}

func TestUpdateKeepsCRLF(t *testing.T) {
	md := "# Titre\r\n\r\ntexte\r\n"
	got := update(md)
	if !strings.Contains(got, "\r\n") {
		t.Error("les fins de ligne CRLF devraient être préservées")
	}
	if strings.Contains(got, "\n\n\n") {
		// un \r\n mal recollé laisserait traîner des \n nus.
		t.Error("ne devrait pas mélanger les styles de fin de ligne")
	}
}
