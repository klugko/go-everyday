package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// jour fige une date pour que les formats restent prévisibles dans les tests.
var jour = time.Date(2026, 6, 8, 14, 32, 0, 0, time.UTC)

func TestNoteFile(t *testing.T) {
	got := noteFile("/journal", jour)
	want := "2026-06-08.md"
	if !strings.HasSuffix(got, want) {
		t.Errorf("noteFile = %q, devrait finir par %q", got, want)
	}
}

func TestEntry(t *testing.T) {
	got := entry(jour, "acheter du café")
	if got != "## 14:32\n\nacheter du café\n\n" {
		t.Errorf("entry inattendue :\n%q", got)
	}
}

func TestHeader(t *testing.T) {
	if got := header(jour); got != "# 2026-06-08\n\n" {
		t.Errorf("header = %q", got)
	}
}

// Le contrat principal : deux notes le même jour atterrissent dans le même
// fichier, sous un seul titre, chacune avec son horodatage.
func TestAppendNote(t *testing.T) {
	dir := t.TempDir()
	path := noteFile(dir, jour)

	if err := appendNote(path, jour, "première"); err != nil {
		t.Fatal(err)
	}
	plusTard := jour.Add(time.Hour)
	if err := appendNote(path, plusTard, "seconde"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if n := strings.Count(got, "# 2026-06-08\n"); n != 1 {
		t.Errorf("le titre du jour devrait apparaître une fois, vu %d :\n%s", n, got)
	}
	for _, want := range []string{"## 14:32", "première", "## 15:32", "seconde"} {
		if !strings.Contains(got, want) {
			t.Errorf("fichier sans %q :\n%s", want, got)
		}
	}
}

func TestRunRejetteVide(t *testing.T) {
	// stdin vide + aucun argument : il n'y a rien à noter, on doit échouer.
	r, w, _ := os.Pipe()
	w.Close()
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	if err := run(t.TempDir(), false, "", jour); err == nil {
		t.Error("une note vide aurait dû être refusée")
	}
}
