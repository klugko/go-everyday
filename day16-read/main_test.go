package main

import (
	"strings"
	"testing"
	"time"
)

func TestSyllables(t *testing.T) {
	cases := map[string]int{
		"chat":       1,
		"bateau":     2, 
		"maison":     2,
		"ordinateur": 4, 
		"a":          1, 
		"42":         1,
	}
	for word, want := range cases {
		if got := syllables(word); got != want {
			t.Errorf("syllables(%q) = %d, attendu %d", word, got, want)
		}
	}
}

func TestCountSentences(t *testing.T) {
	cases := map[string]int{
		"Une phrase.":             1,
		"Une. Deux. Trois.":       3,
		"Vraiment ?! Oui.":        2, 
		"fini...":                 1,
		"aujourd'hui !":           1, 
		"sans ponctuation finale": 0, 
	}
	for text, want := range cases {
		if got := countSentences(text); got != want {
			t.Errorf("countSentences(%q) = %d, attendu %d", text, got, want)
		}
	}
}

func TestAnalyze(t *testing.T) {
	// Deux phrases ; le tiret seul ne doit pas compter comme un mot.
	got := analyze("Le chat dort. — Il fait beau aujourd'hui !")
	if got.Words != 7 {
		t.Errorf("Words = %d, attendu 7", got.Words)
	}
	if got.Sentences != 2 {
		t.Errorf("Sentences = %d, attendu 2", got.Sentences)
	}
}

func TestAnalyzeSansPonctuation(t *testing.T) {
	// Pas de point final : ça reste une phrase, pas zéro (sinon division par zéro).
	got := analyze("juste trois mots")
	if got.Sentences != 1 {
		t.Errorf("Sentences = %d, attendu 1", got.Sentences)
	}
}

func TestFleschOrdonne(t *testing.T) {
	// Un texte court et simple doit scorer plus haut qu'un texte dense et long.
	simple := analyze("Le chat dort. Le chien court. Il fait beau.")
	complexe := analyze("L'inextricable enchevêtrement administratif décourageait irrémédiablement les administrés.")
	if flesch(simple) <= flesch(complexe) {
		t.Errorf("le texte simple (%.0f) devrait scorer plus haut que le complexe (%.0f)",
			flesch(simple), flesch(complexe))
	}
}

func TestReadingTime(t *testing.T) {
	// 200 mots à 200 mots/min = 1 min pile ; 201 doit passer à 2 min (arrondi sup).
	if d := readingTime(200, 200); d != time.Minute {
		t.Errorf("readingTime(200,200) = %s, attendu 1m", d)
	}
	if d := readingTime(201, 200); d != 2*time.Minute {
		t.Errorf("readingTime(201,200) = %s, attendu 2m", d)
	}
	// Même un mot mérite une minute, jamais zéro.
	if d := readingTime(1, 200); d != time.Minute {
		t.Errorf("readingTime(1,200) = %s, attendu 1m", d)
	}
}

func TestFmtDuration(t *testing.T) {
	cases := map[time.Duration]string{
		3 * time.Minute:  "3 min",
		65 * time.Minute: "1 h 05 min",
	}
	for d, want := range cases {
		if got := fmtDuration(d); got != want {
			t.Errorf("fmtDuration(%s) = %q, attendu %q", d, got, want)
		}
	}
}

func TestReportContient(t *testing.T) {
	s := analyze("Le chat dort. Le chien court.")
	out := report(s, 200)
	for _, want := range []string{"mots", "phrases", "temps de lecture", "lisibilité"} {
		if !strings.Contains(out, want) {
			t.Errorf("report sans la ligne %q :\n%s", want, out)
		}
	}
}
