// toc — insère et tient à jour une table des matières dans un fichier Markdown.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// Les marqueurs délimitent la table : tout ce qui est entre eux nous appartient
// et sera réécrit à chaque passage. Le reste du fichier n'est jamais touché.
const (
	openTag  = "<!-- toc -->"
	closeTag = "<!-- /toc -->"
)

func main() {
	write := flag.Bool("w", false, "réécrit le fichier sur place (sinon affiche sur stdout)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "toc — insère et tient à jour une table des matières dans un Markdown.")
		fmt.Fprintln(os.Stderr, "usage : toc [-w] [fichier.md]")
		fmt.Fprintln(os.Stderr, "place les marqueurs", openTag, "/", closeTag, "où tu veux la table ;")
		fmt.Fprintln(os.Stderr, "à défaut elle se glisse après le premier titre. sans fichier, lit stdin.")
		flag.PrintDefaults()
	}
	flag.Parse()

	path := flag.Arg(0)
	if *write && path == "" {
		fmt.Fprintln(os.Stderr, "toc : -w a besoin d'un fichier, il n'y a rien à réécrire sur stdin")
		os.Exit(1)
	}

	md, err := readInput(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "toc :", err)
		os.Exit(1)
	}

	out := update(md)
	if !*write {
		fmt.Print(out)
		return
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "toc :", err)
		os.Exit(1)
	}
}

// readInput : le fichier passé en argument, ou stdin si on n'a rien — pratique en pipe.
func readInput(path string) (string, error) {
	if path == "" {
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

type heading struct {
	level int
	text  string
	slug  string
}

// update reconstruit le document avec une table fraîche. On garde le style de
// fin de ligne d'origine pour ne pas réécrire tout le fichier au passage.
func update(md string) string {
	nl := "\n"
	if strings.Contains(md, "\r\n") {
		nl = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")

	heads := headings(lines)
	if len(heads) == 0 {
		return md // rien à lister, on ne touche à rien
	}
	return strings.Join(splice(lines, toc(heads)), nl)
}

// headings relève les titres ATX (# … ######) en sautant les blocs de code
// clôturés : un "# commentaire" dans du shell n'est pas un titre de section.
func headings(lines []string) []heading {
	var heads []heading
	seen := map[string]int{}
	inCode := false
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if level, text := parseHeading(t); level > 0 {
			heads = append(heads, heading{level, text, slug(text, seen)})
		}
	}
	return heads
}

// parseHeading : "## Titre" → (2, "Titre"). Zéro si ce n'en est pas un — au
// moins un dièse, six au plus, une espace derrière et du texte après.
func parseHeading(s string) (int, string) {
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n >= len(s) || s[n] != ' ' {
		return 0, ""
	}
	text := strings.TrimSpace(s[n+1:])
	if text == "" {
		return 0, ""
	}
	return n, text
}

// slug fabrique l'ancre façon GitHub : minuscules, espaces en tirets, le reste
// de la ponctuation jeté. Deux titres identiques ? on suffixe -1, -2… comme le
// fait GitHub, pour que chaque lien retombe au bon endroit.
func slug(text string, seen map[string]int) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteByte('-')
		}
	}
	s := b.String()
	if n := seen[s]; n > 0 {
		seen[s] = n + 1
		return fmt.Sprintf("%s-%d", s, n)
	}
	seen[s] = 1
	return s
}

// toc rend la liste à puces. L'indentation est relative au titre le moins
// profond présent : une doc qui démarre en ## n'est pas décalée pour rien.
func toc(heads []heading) []string {
	base := heads[0].level
	for _, h := range heads {
		if h.level < base {
			base = h.level
		}
	}
	block := []string{openTag, ""}
	for _, h := range heads {
		indent := strings.Repeat("  ", h.level-base)
		block = append(block, fmt.Sprintf("%s- [%s](#%s)", indent, h.text, h.slug))
	}
	return append(block, "", closeTag)
}

// splice pose le bloc dans le document : il remplace la table existante entre
// les marqueurs si elle est là, sinon il la glisse juste après le premier titre.
func splice(lines, block []string) []string {
	open, end := -1, -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case openTag:
			if open == -1 {
				open = i
			}
		case closeTag:
			end = i
		}
	}

	switch {
	case open != -1 && end > open:
		// table déjà en place : on remplace tout l'entre-deux marqueurs.
		out := append([]string{}, lines[:open]...)
		out = append(out, block...)
		return append(out, lines[end+1:]...)
	case open != -1:
		// un marqueur d'ouverture seul sert de simple point d'insertion.
		out := append([]string{}, lines[:open]...)
		out = append(out, block...)
		return append(out, lines[open+1:]...)
	default:
		// rien de posé : on insère après le premier titre, entouré de blancs.
		at := firstHeading(lines)
		out := append([]string{}, lines[:at+1]...)
		out = append(out, "")
		out = append(out, block...)
		out = append(out, "")
		return append(out, lines[at+1:]...)
	}
}

// firstHeading : l'index du premier titre, blocs de code ignorés. update s'est
// déjà assuré qu'il y en a un, donc le -1 ne devrait jamais sortir.
func firstHeading(lines []string) int {
	inCode := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		if level, _ := parseHeading(t); level > 0 {
			return i
		}
	}
	return -1
}
