package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

func main() {
	find := flag.String("find", "", "motif regex à chercher dans le nom (hors extension)")
	replace := flag.String("replace", "", "remplacement, $1 $2 pour les groupes capturés")
	caseMode := flag.String("case", "", "casse : lower | upper | title")
	numTmpl := flag.String("num", "", "gabarit de numérotation, ex \"{n}-{name}\"")
	start := flag.Int("start", 1, "premier numéro")
	step := flag.Int("step", 1, "pas entre deux numéros")
	pad := flag.Int("pad", 0, "largeur du numéro, 0 = auto (ex 001)")
	glob := flag.String("glob", "", "filtre sur le nom de fichier, ex *.jpg")
	apply := flag.Bool("apply", false, "applique réellement (par défaut : aperçu seulement)")
	flag.Parse()

	r := rules{
		replace:  *replace,
		caseMode: *caseMode,
		numTmpl:  *numTmpl,
		start:    *start,
		step:     *step,
		pad:      *pad,
	}
	if *find != "" {
		re, err := regexp.Compile(*find)
		if err != nil {
			fatal("motif invalide :", err)
		}
		r.find = re
	}
	switch *caseMode {
	case "", "lower", "upper", "title":
	default:
		fatal("case inconnu :", *caseMode, "— attendu lower, upper ou title")
	}
	if r.find == nil && r.caseMode == "" && r.numTmpl == "" {
		fatal("aucune règle : utilise -find/-replace, -case ou -num")
	}

	files, err := gather(flag.Args(), *glob)
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal("aucun fichier à traiter")
	}

	p := buildPlan(files, r)
	render(p, os.Stdout, *apply)

	if len(p.conflicts) > 0 {
		os.Exit(1)
	}
	if *apply && len(p.entries) > 0 {
		fmt.Fprintln(os.Stdout)
		if err := applyPlan(p, os.Stdout); err != nil {
			fatal(err)
		}
	}
}

func fatal(args ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"renamer:"}, args...)...)
	os.Exit(1)
}
