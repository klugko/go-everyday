package main

import (
	"strings"
	"testing"
)

func TestGitignoreBasic(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".gitignore":          "*.log\nnode_modules/\n",
		"app.go":              "package main",
		"debug.log":           "noise",
		"node_modules/foo.js": "lib",
		"src/main.go":         "package main",
	})

	out := renderTree(t, dir, options{useGitignore: true})

	if strings.Contains(out, "debug.log") {
		t.Errorf("debug.log devrait être ignoré, got:\n%s", out)
	}
	if strings.Contains(out, "node_modules") {
		t.Errorf("node_modules/ devrait être ignoré, got:\n%s", out)
	}
	if !strings.Contains(out, "app.go") || !strings.Contains(out, "main.go") {
		t.Errorf("app.go et src/main.go devraient apparaître, got:\n%s", out)
	}
}

func TestGitignoreNegation(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".gitignore": "*.log\n!keep.log\n",
		"drop.log":   "x",
		"keep.log":   "x",
	})

	out := renderTree(t, dir, options{useGitignore: true})

	if strings.Contains(out, "drop.log") {
		t.Errorf("drop.log devrait être ignoré, got:\n%s", out)
	}
	if !strings.Contains(out, "keep.log") {
		t.Errorf("keep.log devrait être ré-inclus par '!keep.log', got:\n%s", out)
	}
}

// un .gitignore enfant doit pouvoir ANNULER
// une règle du parent
func TestNestedGitignore(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".gitignore":        "*.txt\n",
		"top.txt":           "ignored",
		"sub/.gitignore":    "!important.txt\n",
		"sub/important.txt": "kept",
		"sub/other.txt":     "ignored",
	})

	out := renderTree(t, dir, options{useGitignore: true})

	if strings.Contains(out, "top.txt") {
		t.Errorf("top.txt devrait être ignoré, got:\n%s", out)
	}
	if strings.Contains(out, "other.txt") {
		t.Errorf("sub/other.txt devrait toujours être ignoré, got:\n%s", out)
	}
	if !strings.Contains(out, "important.txt") {
		t.Errorf("important.txt devrait être ré-inclus par le .gitignore enfant, got:\n%s", out)
	}
}

func TestDirOnlyPattern(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".gitignore":        "realbuild/\n",
		"build":             "fichier nommé 'build', pas un dossier",
		"realbuild/out.bin": "x",
	})

	out := renderTree(t, dir, options{useGitignore: true})

	if strings.Contains(out, "realbuild") {
		t.Errorf("realbuild/ (dossier) devrait être ignoré, got:\n%s", out)
	}
	if !strings.Contains(out, "build") {
		t.Errorf("le fichier 'build' ne devrait PAS être ignoré par 'realbuild/', got:\n%s", out)
	}
}

func TestDoubleStarPattern(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".gitignore":    "**/*.tmp\n",
		"a.tmp":         "x",
		"sub/b.tmp":     "x",
		"sub/sub/c.tmp": "x",
		"keep.go":       "x",
	})

	out := renderTree(t, dir, options{useGitignore: true})

	for _, name := range []string{"a.tmp", "b.tmp", "c.tmp"} {
		if strings.Contains(out, name) {
			t.Errorf("%s devrait être ignoré par **/*.tmp, got:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "keep.go") {
		t.Errorf("keep.go devrait rester, got:\n%s", out)
	}
}
