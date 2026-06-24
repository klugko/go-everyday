// todoscan — parcourt un projet et liste tous les TODO / FIXME laissés dans le
// code, avec leur position fichier:ligne, prêts à être cliqués dans un éditeur.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	tagsArg := flag.String("tags", "TODO,FIXME", "marqueurs à chercher, séparés par des virgules")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "todoscan — liste les TODO/FIXME d'un projet en fichier:ligne.")
		fmt.Fprintln(os.Stderr, "usage : todoscan [-tags TODO,FIXME,HACK] [chemin]")
		fmt.Fprintln(os.Stderr, "chemin : fichier ou dossier (récursif). défaut : le dossier courant.")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		root = "."
	}
	re, err := buildRE(*tagsArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "todoscan :", err)
		os.Exit(2)
	}

	n, err := scan(root, re, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "todoscan :", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%d marqueur(s) trouvé(s)\n", n)
	if n > 0 {
		os.Exit(1) // pratique en CI : un TODO restant fait échouer le job
	}
}

// hit = un marqueur trouvé, avec de quoi pointer dessus.
type hit struct {
	line int
	tag  string
	text string
}

// buildRE compile un motif depuis la liste de tags. \b…\b évite de matcher TODO
// au milieu d'un mot (« TODOLIST »). On capture le tag et la fin de ligne.
func buildRE(arg string) (*regexp.Regexp, error) {
	var tags []string
	for _, t := range strings.Split(arg, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, regexp.QuoteMeta(t))
		}
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("aucun tag à chercher")
	}
	return regexp.MustCompile(`\b(` + strings.Join(tags, "|") + `)\b(.*)`), nil
}

// scan descend l'arborescence et écrit chaque trouvaille. On saute les dossiers
// cachés et les nids à dépendances (.git, node_modules…) : leurs TODO ne sont
// pas les nôtres.
func scan(root string, re *regexp.Regexp, w io.Writer) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		hits, err := scanFile(p, re)
		if err != nil {
			// Un fichier illisible ne doit pas stopper tout le scan.
			fmt.Fprintf(os.Stderr, "todoscan : %s ignoré (%v)\n", p, err)
			return nil
		}
		for _, h := range hits {
			fmt.Fprintf(w, "%s:%d: %s%s\n", p, h.line, h.tag, h.text)
			count++
		}
		return nil
	})
	return count, err
}

var skipped = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true,
}

// skipDir : on évite les dossiers cachés (commençant par .) et les répertoires
// générés qu'on ne veut jamais fouiller.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipped[name]
}

// scanFile lit un fichier ligne par ligne. On laisse tomber les binaires : un
// octet nul dans le premier bloc trahit une image ou un exécutable.
func scanFile(path string, re *regexp.Regexp) ([]hit, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	if isBinary(br) {
		return nil, nil
	}
	return scanReader(br, re)
}

// scanReader fait le vrai travail : isolé de l'I/O, donc testable directement.
func scanReader(r io.Reader, re *regexp.Regexp) ([]hit, error) {
	var hits []hit
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // tolère les longues lignes minifiées
	for n := 1; sc.Scan(); n++ {
		m := re.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		// m[1] = le tag, m[2] = le reste de la ligne ; on nettoie le « : » et les
		// espaces collés pour ne garder que le message utile.
		text := strings.TrimSpace(m[2])
		text = strings.TrimPrefix(text, ":")
		text = strings.TrimSpace(text)
		if text != "" {
			text = " " + text
		}
		hits = append(hits, hit{line: n, tag: m[1], text: text})
	}
	return hits, sc.Err()
}

// isBinary regarde les 512 premiers octets sans consommer le flux (Peek). Un
// octet nul = contenu non textuel.
func isBinary(br *bufio.Reader) bool {
	buf, _ := br.Peek(512)
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}
	return false
}
