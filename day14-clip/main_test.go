package main

import (
	"strings"
	"testing"
	"time"
)

// base fige une heure de référence pour des âges et des formats prévisibles.
var base = time.Date(2026, 6, 9, 15, 0, 0, 0, time.UTC)

func TestHumanAge(t *testing.T) {
	cas := []struct {
		ecart time.Duration
		want  string
	}{
		{30 * time.Second, "à l'instant"},
		{10 * time.Minute, "il y a 10 min"},
		{3 * time.Hour, "il y a 3 h"},
		{49 * time.Hour, "il y a 2 j"},
	}
	for _, c := range cas {
		if got := humanAge(base, base.Add(-c.ecart)); got != c.want {
			t.Errorf("humanAge(-%s) = %q, attendu %q", c.ecart, got, c.want)
		}
	}
}

func TestPreview(t *testing.T) {
	// Plusieurs lignes et espaces multiples s'aplatissent sur une seule ligne.
	if got := preview("ligne un\n  ligne   deux"); got != "ligne un ligne deux" {
		t.Errorf("preview multi-ligne = %q", got)
	}
	// Au-delà de la limite, on coupe et on ajoute une ellipse.
	long := strings.Repeat("a", 100)
	got := preview(long)
	if !strings.HasSuffix(got, "…") || len([]rune(got)) != 61 {
		t.Errorf("preview long = %q (%d runes)", got, len([]rune(got)))
	}
}

// Le contrat principal de l'écriture : on enregistre, on ignore un doublon
// collé juste après, et on plafonne le nombre d'entrées.
func TestRecord(t *testing.T) {
	path := histFile(t.TempDir())

	if err := record(path, "premier", base); err != nil {
		t.Fatal(err)
	}
	// Recopier la même chose ne doit pas créer de seconde entrée.
	if err := record(path, "premier", base.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := record(path, "second", base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	entries, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("attendu 2 entrées (doublon ignoré), vu %d", len(entries))
	}
	if entries[0].Text != "premier" || entries[1].Text != "second" {
		t.Errorf("ordre inattendu : %q puis %q", entries[0].Text, entries[1].Text)
	}
}

func TestRecordPlafond(t *testing.T) {
	path := histFile(t.TempDir())
	for i := 0; i < maxEntries+10; i++ {
		// Un texte distinct à chaque tour pour éviter la déduplication.
		if err := record(path, string(rune('a'+i%26))+time.Duration(i).String(), base); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != maxEntries {
		t.Errorf("historique plafonné à %d, vu %d", maxEntries, len(entries))
	}
}

func TestLoadRoundTrip(t *testing.T) {
	// Une copie multi-ligne doit survivre à l'aller-retour disque intacte.
	path := histFile(t.TempDir())
	multi := "ligne 1\nligne 2\tavec tab"
	if err := record(path, multi, base); err != nil {
		t.Fatal(err)
	}
	entries, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != multi {
		t.Errorf("aller-retour cassé : %#v", entries)
	}
}

func TestPick(t *testing.T) {
	entries := []entry{
		{At: base, Text: "vieux"},
		{At: base, Text: "récent"},
	}
	// Rang 1 = le plus récent = le dernier enregistré.
	if e, _ := pick(entries, 1); e.Text != "récent" {
		t.Errorf("pick(1) = %q, attendu \"récent\"", e.Text)
	}
	if e, _ := pick(entries, 2); e.Text != "vieux" {
		t.Errorf("pick(2) = %q, attendu \"vieux\"", e.Text)
	}
	if _, err := pick(entries, 3); err == nil {
		t.Error("un rang hors limite aurait dû échouer")
	}
}

func TestFormatList(t *testing.T) {
	if got := formatList(nil, base); !strings.Contains(got, "historique vide") {
		t.Errorf("liste vide = %q", got)
	}

	entries := []entry{
		{At: base.Add(-time.Hour), Text: "ancien"},
		{At: base.Add(-10 * time.Minute), Text: "récent"},
	}
	got := formatList(entries, base)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("attendu 2 lignes, vu %d : %q", len(lines), got)
	}
	// Le plus récent en premier, numéroté 1.
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "1") || !strings.Contains(lines[0], "récent") {
		t.Errorf("première ligne inattendue : %q", lines[0])
	}
}
