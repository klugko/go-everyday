// cloc — compte les lignes de code par langage dans un projet : code,
// commentaires et lignes vides, regroupés et triés.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "cloc — compte les lignes de code par langage.")
		fmt.Fprintln(os.Stderr, "usage : cloc [chemin]")
		fmt.Fprintln(os.Stderr, "chemin : fichier ou dossier (récursif). défaut : le dossier courant.")
	}
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		root = "."
	}
	stats, err := walk(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloc :", err)
		os.Exit(1)
	}
	report(stats, os.Stdout)
}

// stat agrège tout ce qu'on a vu pour un langage donné.
type stat struct {
	lang    string
	files   int
	blank   int
	comment int
	code    int
}

// walk parcourt l'arbre et cumule par langage. On saute les dossiers cachés et
// générés, et tout fichier dont l'extension nous est inconnue.
func walk(root string) (map[string]*stat, error) {
	stats := map[string]*stat{}
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
		l, ok := byExt[strings.ToLower(filepath.Ext(p))]
		if !ok {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cloc : %s ignoré (%v)\n", p, err)
			return nil
		}
		b, c, code := count(string(data), l)

		s := stats[l.name]
		if s == nil {
			s = &stat{lang: l.name}
			stats[l.name] = s
		}
		s.files++
		s.blank += b
		s.comment += c
		s.code += code
		return nil
	})
	return stats, err
}

var skipped = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true,
}

func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipped[name]
}

// report imprime un tableau aligné, langages triés par volume de code, avec une
// ligne de total. Le tri met d'abord ce qui pèse le plus, c'est ce qu'on regarde.
func report(stats map[string]*stat, w *os.File) {
	rows := make([]*stat, 0, len(stats))
	var total stat
	total.lang = "Total"
	for _, s := range stats {
		rows = append(rows, s)
		total.files += s.files
		total.blank += s.blank
		total.comment += s.comment
		total.code += s.code
	}
	if len(rows) == 0 {
		fmt.Fprintln(w, "aucun fichier reconnu")
		return
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].code != rows[j].code {
			return rows[i].code > rows[j].code
		}
		return rows[i].lang < rows[j].lang
	})

	const f = "%-14s %7s %8s %9s %8s\n"
	line := strings.Repeat("-", 50)
	fmt.Fprintln(w, line)
	fmt.Fprintf(w, f, "Langage", "Fichiers", "Vides", "Comm.", "Code")
	fmt.Fprintln(w, line)
	for _, s := range rows {
		fmt.Fprintf(w, f, s.lang, itoa(s.files), itoa(s.blank), itoa(s.comment), itoa(s.code))
	}
	fmt.Fprintln(w, line)
	fmt.Fprintf(w, f, total.lang, itoa(total.files), itoa(total.blank), itoa(total.comment), itoa(total.code))
	fmt.Fprintln(w, line)
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
