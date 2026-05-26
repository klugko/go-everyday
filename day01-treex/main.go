package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	maxDepth := flag.Int("L", 0, "profondeur max (0 = illimité)")
	showHidden := flag.Bool("a", false, "montrer les fichiers cachés (.git reste toujours masqué)")
	noGitignore := flag.Bool("no-gitignore", false, "ne pas lire les .gitignore")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "treex:", err)
		os.Exit(1)
	}

	n := build(abs, nil, options{
		maxDepth:     *maxDepth,
		showHidden:   *showHidden,
		useGitignore: !*noGitignore,
	}, 0)
	if n == nil {
		fmt.Fprintln(os.Stderr, "treex: chemin introuvable")
		os.Exit(1)
	}
	render(os.Stdout, n, "", true, true)
}
