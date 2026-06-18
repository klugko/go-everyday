// faker — génère de fausses données françaises réalistes (noms, emails,
// adresses…) en CSV ou JSON, pour remplir une base de test sans réfléchir.
package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	n := flag.Int("n", 10, "nombre de fiches à générer")
	format := flag.String("f", "csv", "format de sortie : csv | json")
	fieldsArg := flag.String("fields", "prenom,nom,email,tel,ville,cp", "colonnes, séparées par des virgules")
	seed := flag.Int64("seed", 0, "graine aléatoire (0 = aléatoire ; une valeur fixe rejoue le même jeu)")
	out := flag.String("o", "", "fichier de sortie (sortie standard par défaut)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "faker — génère de fausses données FR en CSV ou JSON.")
		fmt.Fprintln(os.Stderr, "usage : faker [-n 10] [-f csv|json] [-fields prenom,email,...] [-seed N] [-o sortie]")
		fmt.Fprintln(os.Stderr, "champs : "+strings.Join(champs, ", "))
		flag.PrintDefaults()
	}
	flag.Parse()

	fields, err := parseFields(*fieldsArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "faker :", err)
		os.Exit(2)
	}

	// Une graine à 0 veut dire « surprends-moi » : on part de l'horloge. Sinon
	// la même graine redonne exactement le même jeu, pratique pour des tests.
	s := *seed
	if s == 0 {
		s = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(s))

	dst := io.Writer(os.Stdout)
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "faker :", err)
			os.Exit(1)
		}
		defer f.Close()
		dst = f
	}

	if err := generate(dst, *format, fields, *n, r); err != nil {
		fmt.Fprintln(os.Stderr, "faker :", err)
		os.Exit(1)
	}
}

// parseFields découpe et valide la liste de colonnes. On refuse tout de suite
// un champ inconnu plutôt que de cracher une colonne vide en silence.
func parseFields(arg string) ([]string, error) {
	known := map[string]bool{}
	for _, c := range champs {
		known[c] = true
	}
	var fields []string
	for _, f := range strings.Split(arg, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !known[f] {
			return nil, fmt.Errorf("champ inconnu %q (au choix : %s)", f, strings.Join(champs, ", "))
		}
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("aucun champ à générer")
	}
	return fields, nil
}

// generate produit n fiches et les écrit dans le format demandé.
func generate(dst io.Writer, format string, fields []string, n int, r *rand.Rand) error {
	switch strings.ToLower(format) {
	case "csv":
		return writeCSV(dst, fields, n, r)
	case "json":
		return writeJSON(dst, fields, n, r)
	default:
		return fmt.Errorf("format inconnu %q (csv ou json)", format)
	}
}

// writeCSV écrit l'en-tête puis une ligne par fiche.
func writeCSV(dst io.Writer, fields []string, n int, r *rand.Rand) error {
	w := csv.NewWriter(dst)
	if err := w.Write(fields); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		p := newPerson(r)
		rec := make([]string, len(fields))
		for j, f := range fields {
			rec[j] = field(p, f, r)
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// writeJSON sort un tableau d'objets, une clé par champ. On garde l'ordre des
// colonnes demandé en écrivant nous-mêmes les paires (une map Go trierait les
// clés), tout en laissant json.Marshal échapper proprement chaque valeur.
func writeJSON(dst io.Writer, fields []string, n int, r *rand.Rand) error {
	bw := bufio.NewWriter(dst)
	bw.WriteString("[\n")
	for i := 0; i < n; i++ {
		p := newPerson(r)
		bw.WriteString("  {")
		for j, f := range fields {
			if j > 0 {
				bw.WriteString(", ")
			}
			k, _ := json.Marshal(f)
			v, _ := json.Marshal(field(p, f, r))
			bw.Write(k)
			bw.WriteString(": ")
			bw.Write(v)
		}
		bw.WriteString("}")
		if i < n-1 {
			bw.WriteString(",")
		}
		bw.WriteString("\n")
	}
	bw.WriteString("]\n")
	return bw.Flush()
}
