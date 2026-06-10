// gign — fabrique un .gitignore pour ta stack à partir de templates embarqués.
package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Les templates voyagent dans le binaire : pas de fichiers à installer à côté,
// gign reste un exécutable autonome qu'on copie où on veut.
//
//go:embed templates/*.gitignore
var templatesFS embed.FS

// alias : les noms qu'on tape par réflexe, ramenés au nom du template.
var alias = map[string]string{
	"golang":   "go",
	"js":       "node",
	"nodejs":   "node",
	"py":       "python",
	"vsc":      "vscode",
	"code":     "vscode",
	"idea":     "jetbrains",
	"intellij": "jetbrains",
	"goland":   "jetbrains",
	"win":      "windows",
	"mac":      "macos",
	"osx":      "macos",
}

func main() {
	list := flag.Bool("list", false, "liste les stacks disponibles")
	out := flag.String("o", "", "écrit dans ce fichier au lieu de la sortie standard")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "gign — un .gitignore pour ta stack, à partir de templates.")
		fmt.Fprintln(os.Stderr, "usage : gign [-o fichier] stack...")
		fmt.Fprintln(os.Stderr, "exemple : gign go node vscode  /  gign -o .gitignore go node")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(flag.Args(), *list, *out); err != nil {
		fmt.Fprintln(os.Stderr, "gign :", err)
		os.Exit(1)
	}
}

func run(stacks []string, list bool, out string) error {
	if list {
		fmt.Println("stacks disponibles :", strings.Join(available(), ", "))
		return nil
	}
	if len(stacks) == 0 {
		flag.Usage()
		return fmt.Errorf("précise au moins une stack (voir --list)")
	}

	content, err := build(stacks)
	if err != nil {
		return err
	}

	if out == "" {
		fmt.Print(content)
		return nil
	}
	if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "écrit dans %s\n", out)
	return nil
}

// section : un template résolu, prêt à être posé sous son titre.
type section struct {
	name string 
	body string
}

// build résout chaque stack demandée puis assemble le tout. Un nom inconnu
// arrête net plutôt que de produire un .gitignore à trous sans le dire.
func build(stacks []string) (string, error) {
	var secs []section
	seenName := map[string]bool{}
	for _, s := range stacks {
		name := resolve(s)
		if seenName[name] {
			continue 
		}
		body, err := templatesFS.ReadFile("templates/" + name + ".gitignore")
		if err != nil {
			return "", fmt.Errorf("stack inconnue %q — voir 'gign --list'", s)
		}
		seenName[name] = true
		secs = append(secs, section{name: name, body: string(body)})
	}
	return combine(secs), nil
}

// resolve normalise un nom de stack : minuscules, puis passage par les alias.
func resolve(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if a, ok := alias[s]; ok {
		return a
	}
	return s
}

// combine empile les sections sous leur titre en dédupliquant les motifs : si
// node et python ignorent tous deux build/, la règle n'apparaît qu'une fois.
// Les commentaires et les lignes vides ne comptent pas comme des doublons.
func combine(secs []section) string {
	var b strings.Builder
	seen := map[string]bool{}

	for _, sec := range secs {
		fmt.Fprintf(&b, "### %s ###\n", title(sec.name))
		blank := false 
		for _, line := range strings.Split(sec.body, "\n") {
			line = strings.TrimRight(line, "\r ")
			switch {
			case line == "":
				if !blank {
					b.WriteString("\n")
					blank = true
				}
			case strings.HasPrefix(line, "#"):
				b.WriteString(line + "\n")
				blank = false
			case seen[line]:
				// motif déjà posé par une section précédente : on saute.
			default:
				seen[line] = true
				b.WriteString(line + "\n")
				blank = false
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// available liste les noms de templates embarqués, triés pour un affichage stable.
func available() []string {
	entries, _ := templatesFS.ReadDir("templates")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
	}
	sort.Strings(names)
	return names
}

// title met juste la première lettre en capitale, assez pour un titre de section.
func title(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}
