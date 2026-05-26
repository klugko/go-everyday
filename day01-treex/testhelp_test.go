package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkTree crée une arborescence
// Un chemin terminé par "/" est un dossier vide.
func mkTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// renderTree construit + rend, et retourne la sortie texte.
func renderTree(t *testing.T, root string, opts options) string {
	t.Helper()
	n := build(root, nil, opts, 0)
	if n == nil {
		t.Fatal("build returned nil")
	}
	var buf bytes.Buffer
	render(&buf, n, "", true, true)
	return buf.String()
}
