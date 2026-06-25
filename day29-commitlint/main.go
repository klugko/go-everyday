// commitlint — valide un message de commit conventionnel (type(scope): sujet)
// et, en mode -make, en compose un bien formé à partir de quelques options.
// Pensé pour servir de hook commit-msg : silencieux si tout va bien, code 1
// sinon.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

func main() {
	typesArg := flag.String("types", "feat,fix,docs,style,refactor,perf,test,build,ci,chore,revert",
		"types autorisés, séparés par des virgules")
	max := flag.Int("max", 72, "longueur maximale du sujet")

	mk := flag.Bool("make", false, "compose un sujet conforme depuis -type/-scope/-desc")
	mType := flag.String("type", "", "(avec -make) type du commit, ex. feat")
	mScope := flag.String("scope", "", "(avec -make) scope, optionnel")
	mDesc := flag.String("desc", "", "(avec -make) description")
	mBreak := flag.Bool("breaking", false, "(avec -make) marque un changement cassant (!)")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "commitlint — valide un message de commit conventionnel (et aide à l'écrire).")
		fmt.Fprintln(os.Stderr, "usage : commitlint [fichier]       # défaut : lit l'entrée standard")
		fmt.Fprintln(os.Stderr, `        commitlint -make -type feat -scope day29 -desc "..."`)
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg := config{types: parseTypes(*typesArg), max: *max}

	// Mode aide à la rédaction : on fabrique le sujet, on le passe au même
	// linter, et on ne l'imprime que s'il est propre.
	if *mk {
		h := buildHeader(*mType, *mScope, *mDesc, *mBreak)
		if probs := lint(h, cfg); len(probs) > 0 {
			report(probs, false)
			os.Exit(2)
		}
		fmt.Println(h)
		return
	}

	msg, err := readMessage(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "commitlint :", err)
		os.Exit(2)
	}
	probs := lint(msg, cfg)
	if len(probs) == 0 {
		return // conforme : on se tait, c'est ce qu'un hook attend
	}
	report(probs, true)
	os.Exit(1)
}

type config struct {
	types map[string]bool
	max   int
}

// list renvoie les types autorisés triés, pour les messages d'erreur.
func (c config) list() string {
	ts := make([]string, 0, len(c.types))
	for t := range c.types {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	return strings.Join(ts, ", ")
}

// La structure d'un sujet conventionnel : type, scope optionnel, « ! » optionnel
// pour un breaking change, puis « : » suivi d'une espace et de la description.
var headerRE = regexp.MustCompile(`^([A-Za-z]+)(\(([^)]*)\))?(!)?: (.*)$`)

// lint renvoie la liste des problèmes du message. Vide = conforme.
func lint(msg string, cfg config) []string {
	lines := clean(msg)
	if len(lines) == 0 {
		return []string{"message de commit vide"}
	}
	probs := lintHeader(lines[0], cfg)
	// Une deuxième ligne non vide colle le corps au sujet : il faut les séparer.
	if len(lines) >= 2 && strings.TrimSpace(lines[1]) != "" {
		probs = append(probs, "il manque une ligne vide entre le sujet et le corps")
	}
	return probs
}

// lintHeader vérifie la première ligne, là où se joue l'essentiel.
func lintHeader(h string, cfg config) []string {
	var probs []string
	if n := len([]rune(h)); n > cfg.max {
		probs = append(probs, fmt.Sprintf("sujet trop long : %d caractères (max %d)", n, cfg.max))
	}

	m := headerRE.FindStringSubmatch(h)
	if m == nil {
		// On distingue le « : » manquant du reste : c'est l'erreur la plus courante.
		if !strings.Contains(h, ": ") {
			probs = append(probs, "format « type(scope): description » — il manque le « : » (avec une espace après)")
		} else {
			probs = append(probs, "format « type(scope): description » — le type doit être un seul mot")
		}
		return probs
	}

	typ, hasScope, scope, desc := m[1], m[2] != "", m[3], strings.TrimRight(m[5], " ")
	switch {
	case !cfg.types[strings.ToLower(typ)]:
		probs = append(probs, fmt.Sprintf("type « %s » inconnu — autorisés : %s", typ, cfg.list()))
	case typ != strings.ToLower(typ):
		probs = append(probs, fmt.Sprintf("type « %s » doit être en minuscules", typ))
	}
	if hasScope && strings.TrimSpace(scope) == "" {
		probs = append(probs, "scope vide entre parenthèses — enlève-les ou nomme le scope")
	}
	switch {
	case desc == "":
		probs = append(probs, "description vide après le « : »")
	case strings.HasSuffix(desc, "."):
		probs = append(probs, "la description ne devrait pas finir par un point")
	}
	return probs
}

// clean retire les lignes de commentaire que git ajoute (# …) et les lignes
// vides en tête, pour ne garder que le vrai message.
func clean(msg string) []string {
	var lines []string
	for _, l := range strings.Split(msg, "\n") {
		l = strings.TrimRight(l, "\r")
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		lines = append(lines, l)
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	return lines
}

// buildHeader assemble un sujet conventionnel ; il sera relu par lint avant
// d'être imprimé, donc pas besoin de valider ici.
func buildHeader(typ, scope, desc string, breaking bool) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(typ))
	if s := strings.TrimSpace(scope); s != "" {
		b.WriteString("(" + s + ")")
	}
	if breaking {
		b.WriteByte('!')
	}
	b.WriteString(": " + strings.TrimSpace(desc))
	return b.String()
}

func parseTypes(arg string) map[string]bool {
	m := map[string]bool{}
	for _, t := range strings.Split(arg, ",") {
		if t = strings.TrimSpace(strings.ToLower(t)); t != "" {
			m[t] = true
		}
	}
	return m
}

// readMessage lit le message depuis un fichier (l'argument d'un hook commit-msg)
// ou, à défaut, depuis l'entrée standard.
func readMessage(path string) (string, error) {
	if path == "" {
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// report imprime les problèmes sur stderr ; withHelp ajoute un rappel du format
// et un exemple — la partie « aide à rédiger ».
func report(probs []string, withHelp bool) {
	fmt.Fprintln(os.Stderr, "commitlint : message non conforme —")
	for _, p := range probs {
		fmt.Fprintln(os.Stderr, "  -", p)
	}
	if withHelp {
		fmt.Fprintln(os.Stderr, "\nformat  : type(scope): description")
		fmt.Fprintln(os.Stderr, "exemple : feat(day29): commitlint, valide les messages de commit")
	}
}
