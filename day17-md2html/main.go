// md2html — convertit un Markdown en page HTML autonome, CSS compris.
package main

import (
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	out := flag.String("o", "", "fichier de sortie (stdout par défaut)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "md2html — transforme un Markdown en page HTML prête à envoyer.")
		fmt.Fprintln(os.Stderr, "usage : md2html [-o sortie.html] [fichier.md]")
		fmt.Fprintln(os.Stderr, "sans fichier, lit l'entrée standard.")
		flag.PrintDefaults()
	}
	flag.Parse()

	md, err := readInput(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "md2html :", err)
		os.Exit(1)
	}

	doc := page(title(md, flag.Arg(0)), convert(md))
	if *out == "" {
		fmt.Print(doc)
		return
	}
	if err := os.WriteFile(*out, []byte(doc), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "md2html :", err)
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

// title cherche le premier "# Titre" ; à défaut le nom du fichier, à défaut
// un libellé neutre. Une page sans <title> fait mauvais effet en pièce jointe.
func title(md, path string) string {
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	if path != "" {
		base := filepath.Base(path)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	return "Document"
}

// convert transforme le corps Markdown en HTML, bloc par bloc. Un parseur
// ligne à ligne suffit largement pour le Markdown qu'on écrit à la main.
func convert(md string) string {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var b strings.Builder
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "":
			i++
		case strings.HasPrefix(line, "```"):
			i = codeBlock(&b, lines, i)
		case isHeading(line):
			level := strings.IndexByte(line, ' ') 
			fmt.Fprintf(&b, "<h%d>%s</h%d>\n", level, inline(strings.TrimSpace(line[level:])), level)
			i++
		case line == "---" || line == "***":
			b.WriteString("<hr>\n")
			i++
		case strings.HasPrefix(line, ">"):
			i = blockquote(&b, lines, i)
		case isItem(line):
			i = list(&b, lines, i)
		default:
			i = paragraph(&b, lines, i)
		}
	}
	return b.String()
}

// isHeading : "## Titre" oui, "##sans-espace" non. Au-delà de six dièses,
// ce n'est plus un titre.
func isHeading(s string) bool {
	n := strings.IndexFunc(s, func(r rune) bool { return r != '#' })
	return n >= 1 && n <= 6 && strings.HasPrefix(s[n:], " ")
}

var reOrdered = regexp.MustCompile(`^\d+\. `)

func isItem(s string) bool {
	return strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") || reOrdered.MatchString(s)
}

// isBlockStart : tout ce qui interrompt un paragraphe en cours.
func isBlockStart(s string) bool {
	return s == "" || s == "---" || s == "***" || isHeading(s) || isItem(s) ||
		strings.HasPrefix(s, ">") || strings.HasPrefix(s, "```")
}

// codeBlock recopie tout jusqu'au ``` fermant : échappé, jamais interprété.
// L'indentation d'origine est gardée telle quelle, c'est du code.
func codeBlock(b *strings.Builder, lines []string, i int) int {
	lang := strings.TrimPrefix(strings.TrimSpace(lines[i]), "```")
	if lang != "" {
		fmt.Fprintf(b, "<pre><code class=%q>", "language-"+lang)
	} else {
		b.WriteString("<pre><code>")
	}
	i++
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != "```"; i++ {
		b.WriteString(html.EscapeString(lines[i]) + "\n")
	}
	b.WriteString("</code></pre>\n")
	if i < len(lines) {
		i++ // saute le ``` fermant
	}
	return i
}

// blockquote regroupe les lignes "> ..." consécutives en une seule citation.
func blockquote(b *strings.Builder, lines []string, i int) int {
	var quote []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(t, ">") {
			break
		}
		quote = append(quote, strings.TrimSpace(strings.TrimPrefix(t, ">")))
	}
	b.WriteString("<blockquote><p>" + inline(strings.Join(quote, " ")) + "</p></blockquote>\n")
	return i
}

// list consomme les puces consécutives. Pas d'imbrication : les listes plates
// couvrent l'essentiel de ce qu'on met dans un mail ou un README.
func list(b *strings.Builder, lines []string, i int) int {
	tag := "ul"
	if reOrdered.MatchString(strings.TrimSpace(lines[i])) {
		tag = "ol"
	}
	fmt.Fprintf(b, "<%s>\n", tag)
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if !isItem(t) {
			break
		}
		t = strings.TrimPrefix(strings.TrimPrefix(t, "- "), "* ")
		t = reOrdered.ReplaceAllString(t, "")
		fmt.Fprintf(b, "<li>%s</li>\n", inline(t))
	}
	fmt.Fprintf(b, "</%s>\n", tag)
	return i
}

// paragraph avale les lignes jusqu'au prochain bloc. Un simple retour à la
// ligne vaut une espace, comme en vrai Markdown.
func paragraph(b *strings.Builder, lines []string, i int) int {
	var p []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if isBlockStart(t) {
			break
		}
		p = append(p, t)
	}
	b.WriteString("<p>" + inline(strings.Join(p, " ")) + "</p>\n")
	return i
}

// inline gère la mise en forme dans une ligne. On isole d'abord les `code`,
// parce que rien ne doit être interprété dedans — ni le gras, ni les liens.
// Split sur les backticks : les segments impairs sont dans le code.
func inline(s string) string {
	parts := strings.Split(s, "`")
	if len(parts)%2 == 0 { 
		parts[len(parts)-2] += "`" + parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}
	var b strings.Builder
	for i, p := range parts {
		if i%2 == 1 {
			b.WriteString("<code>" + html.EscapeString(p) + "</code>")
			continue
		}
		b.WriteString(spans(html.EscapeString(p)))
	}
	return b.String()
}

var (
	reLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reBold   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reItalic = regexp.MustCompile(`\*([^*]+)\*`)
)

// spans : liens d'abord (leur texte peut contenir des étoiles), puis gras
// avant italique, sinon ** serait mangé comme deux étoiles seules.
func spans(s string) string {
	s = reLink.ReplaceAllString(s, `<a href="$2">$1</a>`)
	s = reBold.ReplaceAllString(s, "<strong>$1</strong>")
	s = reItalic.ReplaceAllString(s, "<em>$1</em>")
	return s
}

// css : juste de quoi rendre la lecture agréable — largeur de colonne, police
// système, un peu d'air. Intégré dans la page pour qu'elle voyage seule.
const css = `body {
  max-width: 70ch;
  margin: 2rem auto;
  padding: 0 1rem;
  font-family: system-ui, sans-serif;
  line-height: 1.6;
  color: #222;
}
h1, h2, h3 { line-height: 1.25; }
code {
  background: #f2f2f2;
  padding: 0.15em 0.35em;
  border-radius: 4px;
  font-size: 0.9em;
}
pre {
  background: #f6f6f6;
  padding: 1rem;
  border-radius: 6px;
  overflow-x: auto;
}
pre code { background: none; padding: 0; }
blockquote {
  margin: 0;
  padding-left: 1rem;
  border-left: 3px solid #ccc;
  color: #555;
}
a { color: #0b62c4; }
hr { border: 0; border-top: 1px solid #ddd; }
`

// page emballe le corps dans un document complet : charset, viewport, titre,
// style. Rien d'externe à charger, le fichier s'ouvre partout tel quel.
func page(titre, body string) string {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"fr\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + html.EscapeString(titre) + "</title>\n")
	b.WriteString("<style>\n" + css + "</style>\n</head>\n<body>\n")
	b.WriteString(body)
	b.WriteString("</body>\n</html>\n")
	return b.String()
}
