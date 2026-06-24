package main

import (
	"strings"
	"testing"
)

func TestApplyAddsHeader(t *testing.T) {
	header := render("//", []string{"Copyright (c) 2026 Alice", "SPDX-License-Identifier: MIT"})
	src := "package main\n\nfunc main() {}\n"
	out, did := apply(src, header, spdxMarker)
	if !did {
		t.Fatal("en-tête non ajouté")
	}
	if !strings.HasPrefix(out, "// Copyright (c) 2026 Alice\n// SPDX-License-Identifier: MIT\n\npackage main") {
		t.Fatalf("en-tête mal placé :\n%s", out)
	}
}

// Idempotence : un deuxième passage ne doit rien ajouter.
func TestApplyIdempotent(t *testing.T) {
	header := render("//", []string{"Copyright (c) 2026 Alice", "SPDX-License-Identifier: MIT"})
	src := "package main\n"
	once, _ := apply(src, header, spdxMarker)
	twice, did := apply(once, header, spdxMarker)
	if did {
		t.Fatal("en-tête ajouté deux fois")
	}
	if once != twice {
		t.Fatal("contenu modifié au second passage")
	}
}

// Le shebang doit rester en toute première ligne.
func TestApplyKeepsShebang(t *testing.T) {
	header := render("#", []string{"SPDX-License-Identifier: MIT"})
	src := "#!/usr/bin/env bash\necho hi\n"
	out, did := apply(src, header, spdxMarker)
	if !did {
		t.Fatal("en-tête non ajouté")
	}
	if !strings.HasPrefix(out, "#!/usr/bin/env bash\n") {
		t.Fatalf("shebang déplacé :\n%s", out)
	}
	if !strings.Contains(out, "# SPDX-License-Identifier: MIT") {
		t.Fatalf("en-tête absent après le shebang :\n%s", out)
	}
}

func TestRenderBlankLine(t *testing.T) {
	got := render("//", []string{"a", "", "b"})
	want := "// a\n//\n// b\n"
	if got != want {
		t.Errorf("render = %q, attendu %q", got, want)
	}
}

func TestBuildHeaderMIT(t *testing.T) {
	lines, marker, err := buildHeader("mit", "Bob", 2026, "")
	if err != nil {
		t.Fatal(err)
	}
	if marker != spdxMarker {
		t.Errorf("marqueur = %q", marker)
	}
	if lines[0] != "Copyright (c) 2026 Bob" || lines[1] != "SPDX-License-Identifier: MIT" {
		t.Errorf("en-tête inattendu : %v", lines)
	}
}

func TestBuildHeaderNeedsName(t *testing.T) {
	if _, _, err := buildHeader("mit", "  ", 2026, ""); err == nil {
		t.Error("nom vide accepté")
	}
}

func TestBuildHeaderUnknownLicense(t *testing.T) {
	if _, _, err := buildHeader("wtfpl", "Bob", 2026, ""); err == nil {
		t.Error("licence inconnue acceptée")
	}
}

func TestUnknownExtensionUntouched(t *testing.T) {
	// On vérifie juste la table : une extension absente ne doit pas avoir de
	// préfixe, donc run() la sautera.
	if _, ok := commentPrefix[".png"]; ok {
		t.Error(".png ne devrait pas avoir de style de commentaire")
	}
	if _, ok := commentPrefix[".go"]; !ok {
		t.Error(".go devrait être géré")
	}
}
