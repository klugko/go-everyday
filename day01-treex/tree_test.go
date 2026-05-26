package main

import (
	"strings"
	"testing"
)

func TestSizeAggregation(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		"a.txt":     strings.Repeat("a", 100),
		"sub/b.txt": strings.Repeat("b", 250),
		"sub/c.txt": strings.Repeat("c", 50),
	})

	n := build(dir, nil, options{useGitignore: false}, 0)
	if n.size != 400 {
		t.Errorf("root size = %d, want 400", n.size)
	}

	var sub *node
	for _, c := range n.children {
		if c.name == "sub" {
			sub = c
		}
	}
	if sub == nil {
		t.Fatal("sub introuvable")
	}
	if sub.size != 300 {
		t.Errorf("sub size = %d, want 300", sub.size)
	}
}

func TestGitFolderAlwaysSkipped(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".git/HEAD":   "ref: refs/heads/main",
		".git/config": "[core]",
		"main.go":     "package main",
	})

	out := renderTree(t, dir, options{showHidden: true, useGitignore: false})
	if strings.Contains(out, ".git") {
		t.Errorf(".git/ devrait toujours être masqué, got:\n%s", out)
	}
}

func TestHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		".env":   "SECRET=x",
		"app.go": "package main",
	})

	if out := renderTree(t, dir, options{}); strings.Contains(out, ".env") {
		t.Errorf(".env devrait être caché par défaut, got:\n%s", out)
	}
	if out := renderTree(t, dir, options{showHidden: true}); !strings.Contains(out, ".env") {
		t.Errorf("avec -a, .env devrait apparaître, got:\n%s", out)
	}
}

func TestMaxDepth(t *testing.T) {
	dir := t.TempDir()
	mkTree(t, dir, map[string]string{
		"a/b/c/deep.txt": "x",
		"top.txt":        "x",
	})

	out := renderTree(t, dir, options{maxDepth: 1})
	if !strings.Contains(out, "top.txt") {
		t.Errorf("top.txt devrait apparaître à -L 1, got:\n%s", out)
	}
	if strings.Contains(out, "deep.txt") {
		t.Errorf("deep.txt ne devrait PAS apparaître avec -L 1, got:\n%s", out)
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1024 * 1024, "1.0 MiB"},
	}
	for _, c := range cases {
		if got := humanSize(c.in); got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
