package main

import "strings"

// Le marqueur SPDX sert deux fois : il identifie la licence de façon standard
// (outils de conformité, GitHub…) et il nous sert de sentinelle pour ne jamais
// réinsérer un en-tête déjà présent.
const spdxMarker = "SPDX-License-Identifier"

// Identifiants SPDX des licences proposées en dur. Le reste passe par -text.
var spdxID = map[string]string{
	"mit":        "MIT",
	"apache-2.0": "Apache-2.0",
	"gpl-3.0":    "GPL-3.0-or-later",
	"bsd-3":      "BSD-3-Clause",
}

// Préfixe de commentaire de ligne par extension. Pas d'entrée = on ne sait pas
// commenter ce fichier, donc on n'y touche pas.
var commentPrefix = map[string]string{
	".go": "//", ".c": "//", ".h": "//", ".cpp": "//", ".cc": "//",
	".java": "//", ".js": "//", ".mjs": "//", ".ts": "//", ".tsx": "//", ".rs": "//",
	".py": "#", ".rb": "#", ".sh": "#", ".bash": "#", ".yml": "#", ".yaml": "#",
}

// render transforme des lignes de texte brut en bloc de commentaire. Une ligne
// vide reste vide (juste le préfixe), pour aérer sans laisser de blanc parasite.
func render(prefix string, lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		if l == "" {
			b.WriteString(prefix + "\n")
		} else {
			b.WriteString(prefix + " " + l + "\n")
		}
	}
	return b.String()
}

// apply ajoute le bloc en tête du fichier, sauf s'il y est déjà (marker). Le
// shebang, s'il existe, doit rester en toute première ligne : on glisse l'en-tête
// juste après. Renvoie le nouveau contenu et s'il a changé.
func apply(content, header, marker string) (string, bool) {
	if alreadyHas(content, marker) {
		return content, false
	}
	if strings.HasPrefix(content, "#!") {
		nl := strings.IndexByte(content, '\n')
		if nl < 0 {
			// Fichier d'une seule ligne de shebang, sans saut final.
			return content + "\n\n" + header, true
		}
		shebang, rest := content[:nl+1], content[nl+1:]
		return shebang + header + "\n" + rest, true
	}
	return header + "\n" + content, true
}

// alreadyHas cherche le marqueur dans le haut du fichier seulement : un SPDX
// cité plus bas (dans une string, un test…) ne doit pas faire croire que
// l'en-tête est posé.
func alreadyHas(content, marker string) bool {
	head := content
	if len(head) > 1024 {
		head = head[:1024]
	}
	return strings.Contains(head, marker)
}
