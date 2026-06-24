package main

import "testing"

func TestCountGo(t *testing.T) {
	src := `package main

// un commentaire de ligne
func main() {
	/* bloc
	   sur deux lignes */
	x := 1 // code avec commentaire en bout → compte pour du code
}
`
	l := byExt[".go"]
	blank, comment, code := count(src, l)
	if blank != 1 {
		t.Errorf("vides = %d, attendu 1", blank)
	}
	if comment != 3 { // la ligne //, et les deux lignes du bloc
		t.Errorf("commentaires = %d, attendu 3", comment)
	}
	if code != 4 { // package, func, x:=1, }
		t.Errorf("code = %d, attendu 4", code)
	}
}

func TestBlockOnOneLine(t *testing.T) {
	l := byExt[".go"]
	_, comment, code := count("/* tout sur une ligne */\ncode()\n", l)
	if comment != 1 {
		t.Errorf("commentaires = %d, attendu 1", comment)
	}
	if code != 1 {
		t.Errorf("code = %d, attendu 1", code)
	}
}

func TestHashStyle(t *testing.T) {
	l := byExt[".py"]
	blank, comment, code := count("# titre\nprint('x')\n\n", l)
	if blank != 1 || comment != 1 || code != 1 {
		t.Errorf("py : vides=%d comm=%d code=%d, attendu 1/1/1", blank, comment, code)
	}
}

// Pas de saut de ligne final : la dernière ligne doit quand même compter.
func TestNoTrailingNewline(t *testing.T) {
	l := byExt[".go"]
	_, _, code := count("a()\nb()", l)
	if code != 2 {
		t.Errorf("code = %d, attendu 2", code)
	}
}

// Un saut de ligne final ne doit pas inventer une ligne vide.
func TestTrailingNewlineNotCounted(t *testing.T) {
	l := byExt[".go"]
	blank, _, code := count("a()\n", l)
	if blank != 0 || code != 1 {
		t.Errorf("vides=%d code=%d, attendu 0/1", blank, code)
	}
}

func TestEmptyFile(t *testing.T) {
	l := byExt[".go"]
	if b, c, code := count("", l); b+c+code != 0 {
		t.Errorf("fichier vide devrait tout donner à 0, obtenu %d/%d/%d", b, c, code)
	}
}

func TestUnterminatedBlock(t *testing.T) {
	// Bloc ouvert et jamais refermé : tout ce qui suit reste du commentaire.
	l := byExt[".go"]
	_, comment, code := count("code()\n/* début\nencore\ntoujours\n", l)
	if code != 1 {
		t.Errorf("code = %d, attendu 1", code)
	}
	if comment != 3 {
		t.Errorf("commentaires = %d, attendu 3", comment)
	}
}
