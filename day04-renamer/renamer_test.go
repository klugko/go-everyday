package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestTransformFindReplaceKeepsExt(t *testing.T) {
	r := rules{find: regexp.MustCompile("IMG_"), replace: "photo_"}
	if got := r.transform("IMG_001.JPG", 0, 1); got != "photo_001.JPG" {
		t.Fatalf("got %q", got)
	}
}

func TestTransformCaseOnlyTouchesStem(t *testing.T) {
	r := rules{caseMode: "lower"}
	// L'extension .PNG ne doit pas passer en minuscules.
	if got := r.transform("PHOTO.PNG", 0, 1); got != "photo.PNG" {
		t.Fatalf("got %q", got)
	}
}

func TestTransformTitle(t *testing.T) {
	r := rules{caseMode: "title"}
	if got := r.transform("my_holiday-2024.txt", 0, 1); got != "My_Holiday-2024.txt" {
		t.Fatalf("got %q", got)
	}
}

func TestTransformNumberingAutoPad(t *testing.T) {
	r := rules{numTmpl: "{n}-{name}", start: 1, step: 1}
	// 12 fichiers → numéros sur 2 chiffres.
	if got := r.transform("a.jpg", 0, 12); got != "01-a.jpg" {
		t.Fatalf("got %q", got)
	}
	if got := r.transform("k.jpg", 9, 12); got != "10-k.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestTransformNumberingExplicitPadAndStart(t *testing.T) {
	r := rules{numTmpl: "img-{n}", start: 5, step: 5, pad: 3}
	if got := r.transform("x.png", 0, 3); got != "img-005.png" {
		t.Fatalf("got %q", got)
	}
	if got := r.transform("z.png", 2, 3); got != "img-015.png" {
		t.Fatalf("got %q", got)
	}
}

func TestTransformChainsRules(t *testing.T) {
	// find → case → num, dans cet ordre.
	r := rules{
		find: regexp.MustCompile(`\s+`), replace: "_",
		caseMode: "lower", numTmpl: "{n}_{name}", start: 1, pad: 2,
	}
	if got := r.transform("My Photo.JPG", 0, 1); got != "01_my_photo.JPG" {
		t.Fatalf("got %q", got)
	}
}

func TestGatherGlobOnDir(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.jpg"), "x")
	write(t, filepath.Join(root, "b.png"), "x")
	write(t, filepath.Join(root, "c.jpg"), "x")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	files, err := gather([]string{root}, "*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("%d fichiers, attendu 2 : %v", len(files), files)
	}
}

func TestBuildPlanDetectsCollision(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	write(t, a, "1")
	write(t, b, "2")
	// Gabarit constant : a et b visent tous deux "file.txt".
	r := rules{numTmpl: "file"}
	p := buildPlan([]string{a, b}, r)
	if len(p.conflicts) == 0 {
		t.Fatalf("collision non détectée : %+v", p)
	}
}

func TestBuildPlanDetectsExistingTarget(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	write(t, a, "1")
	write(t, b, "2") // existe déjà et ne bouge pas
	r := rules{find: regexp.MustCompile("^a$"), replace: "b"}
	p := buildPlan([]string{a}, r)
	if len(p.conflicts) != 1 {
		t.Fatalf("attendu 1 conflit, got %d : %v", len(p.conflicts), p.conflicts)
	}
}

func TestBuildPlanCountsUnchanged(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "keep.txt")
	write(t, a, "1")
	// find qui ne matche rien donc aucun changement.
	r := rules{find: regexp.MustCompile("zzz"), replace: "y"}
	p := buildPlan([]string{a}, r)
	if len(p.entries) != 0 || p.unchanged != 1 {
		t.Fatalf("entries=%d unchanged=%d", len(p.entries), p.unchanged)
	}
}

func TestApplyPlanRenamesOnDisk(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "old.txt")
	write(t, a, "data")
	r := rules{find: regexp.MustCompile("old"), replace: "new"}
	p := buildPlan([]string{a}, r)

	var buf bytes.Buffer
	if err := applyPlan(p, &buf); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "new.txt")); err != nil {
		t.Fatalf("new.txt absent : %v", err)
	}
	if _, err := os.Stat(a); !os.IsNotExist(err) {
		t.Fatalf("old.txt aurait dû disparaître")
	}
}

func TestApplyPlanHandlesSwap(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	b := filepath.Join(root, "b.txt")
	write(t, a, "AAA")
	write(t, b, "BBB")
	// Échange direct a-b 
	p := plan{entries: []entry{
		{oldPath: a, newPath: b, oldName: "a.txt", newName: "b.txt"},
		{oldPath: b, newPath: a, oldName: "b.txt", newName: "a.txt"},
	}}

	var buf bytes.Buffer
	if err := applyPlan(p, &buf); err != nil {
		t.Fatal(err)
	}
	if got := read(t, a); got != "BBB" {
		t.Fatalf("a.txt = %q, attendu BBB", got)
	}
	if got := read(t, b); got != "AAA" {
		t.Fatalf("b.txt = %q, attendu AAA", got)
	}
}

func TestRenderPreviewMentionsApply(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.txt")
	write(t, a, "1")
	r := rules{caseMode: "upper"}
	p := buildPlan([]string{a}, r)

	var buf bytes.Buffer
	render(p, &buf, false)
	out := buf.String()
	if !strings.Contains(out, "-apply") {
		t.Fatalf("l'aperçu devrait inviter à -apply : %q", out)
	}
	if !strings.Contains(out, "A.txt") {
		t.Fatalf("la cible manque : %q", out)
	}
}

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
