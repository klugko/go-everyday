package main

import (
	"strings"
	"testing"
)

func TestScanReaderFindsTags(t *testing.T) {
	re, err := buildRE("TODO,FIXME")
	if err != nil {
		t.Fatal(err)
	}
	src := `package main
// TODO: gérer le cas vide
func f() {}
// rien ici
x := 1 // FIXME off-by-one
`
	hits, err := scanReader(strings.NewReader(src), re)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("attendu 2 marqueurs, obtenu %d : %+v", len(hits), hits)
	}
	if hits[0].line != 2 || hits[0].tag != "TODO" || hits[0].text != " gérer le cas vide" {
		t.Errorf("premier hit inattendu : %+v", hits[0])
	}
	if hits[1].line != 5 || hits[1].tag != "FIXME" || hits[1].text != " off-by-one" {
		t.Errorf("second hit inattendu : %+v", hits[1])
	}
}

// TODO collé dans un mot ne doit pas matcher : c'est la frontière \b qui protège.
func TestWordBoundary(t *testing.T) {
	re, _ := buildRE("TODO")
	hits, _ := scanReader(strings.NewReader("var TODOLIST = 3\n"), re)
	if len(hits) != 0 {
		t.Errorf("TODOLIST ne devrait pas matcher : %+v", hits)
	}
}

func TestTagWithoutColon(t *testing.T) {
	re, _ := buildRE("HACK")
	hits, _ := scanReader(strings.NewReader("// HACK rustine temporaire\n"), re)
	if len(hits) != 1 || hits[0].text != " rustine temporaire" {
		t.Fatalf("hit inattendu : %+v", hits)
	}
}

func TestCustomTags(t *testing.T) {
	re, err := buildRE("XXX, NOTE")
	if err != nil {
		t.Fatal(err)
	}
	hits, _ := scanReader(strings.NewReader("# XXX revoir\n# NOTE: au passage\n# TODO ignoré\n"), re)
	if len(hits) != 2 {
		t.Fatalf("attendu 2 hits, obtenu %d : %+v", len(hits), hits)
	}
}

func TestEmptyTags(t *testing.T) {
	if _, err := buildRE("  , "); err == nil {
		t.Error("liste de tags vide acceptée")
	}
}

func TestSkipDir(t *testing.T) {
	for _, d := range []string{".git", "node_modules", ".idea", "vendor"} {
		if !skipDir(d) {
			t.Errorf("%q devrait être sauté", d)
		}
	}
	if skipDir("src") {
		t.Error("src ne devrait pas être sauté")
	}
}
