package main

import (
	"reflect"
	"strings"
	"testing"
)

func newCfg() config {
	return config{types: parseTypes("feat,fix,docs,chore"), max: 72}
}

func TestValide(t *testing.T) {
	ok := []string{
		"feat: ajoute le mode hors-ligne",
		"fix(api): corrige le timeout",
		"feat(day29)!: change la signature publique",
		"chore: bump go 1.26",
	}
	cfg := newCfg()
	for _, msg := range ok {
		if p := lint(msg, cfg); len(p) != 0 {
			t.Errorf("%q devrait être valide, obtenu %v", msg, p)
		}
	}
}

func TestTypeInconnu(t *testing.T) {
	p := lint("wip: en cours", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "inconnu") {
		t.Fatalf("type inconnu non détecté : %v", p)
	}
}

func TestTypeMajuscule(t *testing.T) {
	p := lint("Feat: ajoute X", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "minuscules") {
		t.Fatalf("type majuscule non détecté : %v", p)
	}
}

func TestSeparateurManquant(t *testing.T) {
	p := lint("feat ajoute X", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "« : »") && !strings.Contains(p[0], ": ") {
		t.Fatalf("séparateur manquant non détecté : %v", p)
	}
}

func TestDescriptionVide(t *testing.T) {
	p := lint("fix: ", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "vide") {
		t.Fatalf("description vide non détectée : %v", p)
	}
}

func TestPointFinal(t *testing.T) {
	p := lint("docs: corrige le README.", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "point") {
		t.Fatalf("point final non détecté : %v", p)
	}
}

func TestScopeVide(t *testing.T) {
	p := lint("feat(): ajoute X", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "scope") {
		t.Fatalf("scope vide non détecté : %v", p)
	}
}

func TestSujetTropLong(t *testing.T) {
	long := "feat: " + strings.Repeat("x", 80)
	p := lint(long, newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "trop long") {
		t.Fatalf("sujet trop long non détecté : %v", p)
	}
}

func TestLigneVideManquante(t *testing.T) {
	p := lint("feat: ajoute X\nle corps collé au sujet", newCfg())
	if len(p) != 1 || !strings.Contains(p[0], "ligne vide") {
		t.Fatalf("corps collé non détecté : %v", p)
	}
}

func TestCorpsBienSepare(t *testing.T) {
	if p := lint("feat: ajoute X\n\nun corps correct", newCfg()); len(p) != 0 {
		t.Fatalf("ne devrait rien signaler : %v", p)
	}
}

// Les lignes de commentaire de git (#) et les blancs en tête sont ignorés.
func TestCommentairesIgnores(t *testing.T) {
	msg := "# Please enter the commit message\n\nfeat: ajoute X\n"
	if p := lint(msg, newCfg()); len(p) != 0 {
		t.Fatalf("les commentaires git auraient dû être ignorés : %v", p)
	}
}

func TestMessageVide(t *testing.T) {
	if p := lint("\n# que des commentaires\n", newCfg()); len(p) != 1 {
		t.Fatalf("message vide non détecté : %v", p)
	}
}

func TestBuildHeader(t *testing.T) {
	cases := []struct {
		typ, scope, desc string
		breaking         bool
		want             string
	}{
		{"feat", "day29", "ajoute X", false, "feat(day29): ajoute X"},
		{"fix", "", "corrige Y", false, "fix: corrige Y"},
		{"feat", "api", "casse tout", true, "feat(api)!: casse tout"},
	}
	for _, c := range cases {
		got := buildHeader(c.typ, c.scope, c.desc, c.breaking)
		if got != c.want {
			t.Errorf("buildHeader(%q,%q,%q,%v) = %q, attendu %q", c.typ, c.scope, c.desc, c.breaking, got, c.want)
		}
		// Ce qu'on fabrique doit toujours passer le linter.
		if p := lint(got, newCfg()); len(p) != 0 {
			t.Errorf("%q produit par buildHeader est rejeté : %v", got, p)
		}
	}
}

func TestParseTypes(t *testing.T) {
	got := parseTypes(" feat, Fix ,, ")
	want := map[string]bool{"feat": true, "fix": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTypes = %v, attendu %v", got, want)
	}
}
