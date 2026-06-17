// csv2x — exporte un CSV vers JSON, une table Markdown ou un classeur Excel.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	format := flag.String("f", "", "format de sortie : json | md | xlsx (déduit de -o sinon json)")
	out := flag.String("o", "", "fichier de sortie (sortie standard par défaut)")
	delim := flag.String("d", ",", "délimiteur du CSV d'entrée")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "csv2x — convertit un CSV en JSON, table Markdown ou Excel (.xlsx).")
		fmt.Fprintln(os.Stderr, "usage : csv2x [-f json|md|xlsx] [-o sortie] [-d ,] fichier.csv")
		fmt.Fprintln(os.Stderr, "le fichier peut être '-' pour lire l'entrée standard.")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	// Le format vient du -f ; à défaut on le devine de l'extension du -o.
	f := pickFormat(*format, *out)
	if f == "" {
		fmt.Fprintln(os.Stderr, "csv2x : format inconnu (json, md ou xlsx)")
		os.Exit(2)
	}

	header, rows, err := readCSV(flag.Arg(0), []rune(*delim)[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "csv2x :", err)
		os.Exit(1)
	}

	if err := export(f, *out, header, rows); err != nil {
		fmt.Fprintln(os.Stderr, "csv2x :", err)
		os.Exit(1)
	}
}

// pickFormat normalise le format demandé, ou le déduit de l'extension du
// fichier de sortie. Quelques alias parce qu'on hésite toujours entre md et
// markdown, xlsx et excel.
func pickFormat(f, out string) string {
	if f == "" && out != "" {
		f = strings.TrimPrefix(filepath.Ext(out), ".")
	}
	switch strings.ToLower(f) {
	case "", "json":
		return "json"
	case "md", "markdown":
		return "md"
	case "xlsx", "excel":
		return "xlsx"
	default:
		return ""
	}
}

// export branche le bon writer sur la bonne destination. Le .xlsx est binaire :
// le déverser dans un terminal n'a aucun sens, on exige donc un -o.
func export(format, out string, header []string, rows [][]string) error {
	if format == "xlsx" && out == "" {
		return fmt.Errorf("le format xlsx est binaire, précisez un fichier avec -o")
	}

	dst := io.Writer(os.Stdout)
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		dst = f
	}

	switch format {
	case "json":
		return writeJSON(dst, header, rows)
	case "md":
		return writeMarkdown(dst, header, rows)
	case "xlsx":
		return writeXLSX(dst, header, rows)
	}
	return nil
}

// readCSV lit le fichier (ou stdin si '-'), première ligne = en-tête.
func readCSV(path string, sep rune) (header []string, rows [][]string, err error) {
	src := os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		src = f
	}

	r := csv.NewReader(src)
	r.Comma = sep
	r.FieldsPerRecord = -1 

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("fichier vide")
	}
	header = all[0]
	// Un CSV sorti d'Excel traîne souvent un BOM collé au premier en-tête.
	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	return header, all[1:], nil
}

// cell lit une cellule en tolérant les lignes trop courtes.
func cell(row []string, i int) string {
	if i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}
