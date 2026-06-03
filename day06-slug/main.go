package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	sep := flag.String("sep", "-", "séparateur entre les mots")
	max := flag.Int("max", 0, "longueur maximale en octets (0 = sans limite)")
	keepCase := flag.Bool("keep-case", false, "préserver la casse au lieu de tout passer en minuscules")
	flag.Parse()

	opt := Options{Sep: *sep, Lower: !*keepCase, Max: *max}

	// Un titre en argument -> un slug. Sinon on lit l'entrée standard
	// ligne par ligne
	if flag.NArg() > 0 {
		fmt.Println(Slugify(strings.Join(flag.Args(), " "), opt))
		return
	}

	sc := bufio.NewScanner(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for sc.Scan() {
		fmt.Fprintln(w, Slugify(sc.Text(), opt))
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "slug:", err)
		os.Exit(1)
	}
}
