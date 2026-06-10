package main

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	cas := map[string]string{
		"go":      "go",
		"Go":      "go",
		"golang":  "go",
		"  JS  ":  "node",
		"py":      "python",
		"osx":     "macos",
		"inconnu": "inconnu", 
	}
	for in, want := range cas {
		if got := resolve(in); got != want {
			t.Errorf("resolve(%q) = %q, attendu %q", in, got, want)
		}
	}
}

func TestBuildInconnu(t *testing.T) {
	if _, err := build([]string{"cobol"}); err == nil {
		t.Error("une stack inconnue aurait dû échouer")
	}
}

func TestBuildSection(t *testing.T) {
	// Un alias doit bien charger son template et coiffer la section d'un titre.
	out, err := build([]string{"golang"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "### Go ###") {
		t.Errorf("titre de section manquant :\n%s", out)
	}
	if !strings.Contains(out, "*.exe") {
		t.Errorf("motif Go attendu absent :\n%s", out)
	}
}

func TestBuildDoublonStack(t *testing.T) {
	// « gign go go » ne doit pas répéter la section.
	out, err := build([]string{"go", "Go", "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "### Go ###"); n != 1 {
		t.Errorf("section Go présente %d fois, attendu 1", n)
	}
}

func TestCombineDedupMotifs(t *testing.T) {
	// Deux sections partageant un motif : le motif n'apparaît qu'une fois,
	// mais chaque titre reste là.
	secs := []section{
		{name: "node", body: "node_modules/\nbuild/\n"},
		{name: "python", body: "build/\n__pycache__/\n"},
	}
	out := combine(secs)
	if n := strings.Count(out, "build/"); n != 1 {
		t.Errorf("motif partagé 'build/' présent %d fois, attendu 1", n)
	}
	if !strings.Contains(out, "### Node ###") || !strings.Contains(out, "### Python ###") {
		t.Errorf("un titre de section manque :\n%s", out)
	}
	if !strings.Contains(out, "__pycache__/") {
		t.Errorf("motif propre à python perdu :\n%s", out)
	}
}

func TestCombineGardeCommentaires(t *testing.T) {
	// Un commentaire identique dans deux sections n'est pas dédupliqué :
	// c'est un repère de lecture, pas une règle.
	secs := []section{
		{name: "a", body: "# Build\nx\n"},
		{name: "b", body: "# Build\ny\n"},
	}
	out := combine(secs)
	if n := strings.Count(out, "# Build"); n != 2 {
		t.Errorf("commentaire dédupliqué à tort : vu %d fois, attendu 2", n)
	}
}

func TestAvailable(t *testing.T) {
	got := available()
	if len(got) == 0 {
		t.Fatal("aucun template embarqué")
	}
	// Trié et sans l'extension .gitignore.
	for _, n := range got {
		if strings.HasSuffix(n, ".gitignore") {
			t.Errorf("extension non retirée : %q", n)
		}
	}
	if !sortedAsc(got) {
		t.Errorf("liste non triée : %v", got)
	}
}

func sortedAsc(s []string) bool {
	for i := 1; i < len(s); i++ {
		if s[i-1] > s[i] {
			return false
		}
	}
	return true
}
