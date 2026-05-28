package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	interactive := flag.Bool("i", false, "mode interactif : proposer de supprimer les doublons")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dupes:", err)
		os.Exit(1)
	}

	bySize := scan(abs)
	groups := group(bySize, runtime.NumCPU())
	list := sortedGroups(groups)
	render(list, abs, os.Stdout)

	if *interactive && len(list) > 0 {
		prompt(list, abs, os.Stdin, os.Stdout)
	}
}

// propose pour chaque groupe de garder un fichier et supprimer
//  Conventions :
//
//	un numéro  → on garde ce fichier, on supprime les autres
//	"" (entrée)→ on saute ce groupe
//	"q"        → on arrête tout
func prompt(list []*dupGroup, root string, in io.Reader, w io.Writer) {
	r := bufio.NewReader(in)
	fmt.Fprintln(w)
	for i, g := range list {
		fmt.Fprintf(w, "\n[%d/%d] %s × %d copies\n",
			i+1, len(list), humanSize(g.size), len(g.paths))
		for j, p := range g.paths {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				rel = p
			}
			fmt.Fprintf(w, "  %d) %s\n", j+1, rel)
		}
		fmt.Fprint(w, "Garder lequel ? (numéro / entrée pour passer / q pour quitter) :")
		line, err := r.ReadString('\n')
		// EOF avec contenu : on traite quand même. EOF sans contenu : on sort.
		if err != nil && (err != io.EOF || line == "") {
			return
		}
		line = strings.TrimSpace(line)
		switch line {
		case "q":
			return
		case "", "s":
			continue
		}
		keep, err := strconv.Atoi(line)
		if err != nil || keep < 1 || keep > len(g.paths) {
			fmt.Fprintln(w, "  réponse invalide, groupe passé.")
			continue
		}
		for j, p := range g.paths {
			if j+1 == keep {
				continue
			}
			if err := os.Remove(p); err != nil {
				fmt.Fprintf(w, "  ! %s: %v\n", p, err)
			} else {
				fmt.Fprintf(w, "  supprimé %s\n", p)
			}
		}
	}
}
