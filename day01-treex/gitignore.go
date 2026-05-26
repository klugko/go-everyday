package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Une règle compilée. `base` est le dossier où vivait le .gitignore,
// dont sert d'ancrage pour calculer le chemin relatif au moment du match.
type rule struct {
	base    string
	re      *regexp.Regexp
	negate  bool // commence par '!'
	dirOnly bool // se termine par '/'
}

// readGitignore lit le .gitignore d'un dossier (s'il existe).
func readGitignore(dir string) []rule {
	f, err := os.Open(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []rule
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if r := compileRule(dir, sc.Text()); r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// compileRule traduit une ligne de .gitignore en regexp.
// Retourne nil pour les commentaires et lignes vides.
//
// On utilise une regexp plutôt que filepath.Match parce que Match ne sait
// pas faire `**` (segments arbitraires) ni distinguer pattern ancré vs flottant.
func compileRule(base, line string) *rule {
	line = strings.TrimRight(line, " \t")
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	r := rule{base: base}
	if strings.HasPrefix(line, "!") {
		r.negate = true
		line = line[1:]
	}
	if strings.HasSuffix(line, "/") {
		r.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	// Un '/' quelque part dans la ligne = pattern ancré à la racine du
	// .gitignore. Sinon il flotte et peut matcher à n'importe quelle profondeur.
	anchored := strings.Contains(line, "/")
	line = strings.TrimPrefix(line, "/")

	var sb strings.Builder
	sb.WriteString("^")
	if !anchored {
		sb.WriteString(`(?:.*/)?`)
	}
	for i := 0; i < len(line); {
		c := line[i]
		switch {
		case c == '*' && i+1 < len(line) && line[i+1] == '*':
			if i+2 < len(line) && line[i+2] == '/' {
				sb.WriteString(`(?:[^/]+/)*`) // **/ : zéro ou plusieurs segments
				i += 3
			} else {
				sb.WriteString(`.*`) // ** en fin : tout, slash inclus
				i += 2
			}
		case c == '*':
			sb.WriteString(`[^/]*`)
			i++
		case c == '?':
			sb.WriteString(`[^/]`)
			i++
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
			i++
		}
	}
	// On laisse aussi matcher les descendants : si "node_modules" matche le
	// dossier, on veut aussi "node_modules/foo/bar" comme ignoré.
	sb.WriteString(`(?:/.*)?$`)

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return nil
	}
	r.re = re
	return &r
}

// isIgnored applique la liste de règles à un chemin.
//
// Sémantique git : on parcourt dans l'ordre, le DERNIER match l'emporte.
// Les règles d'un .gitignore enfant sont append après celles du parent,
// donc elles gagnent naturellement — pas besoin de stack explicite.
func isIgnored(rules []rule, path string, isDir bool) bool {
	ignored := false
	for _, r := range rules {
		if r.dirOnly && !isDir {
			continue
		}
		rel, err := filepath.Rel(r.base, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if r.re.MatchString(filepath.ToSlash(rel)) {
			ignored = !r.negate
		}
	}
	return ignored
}
