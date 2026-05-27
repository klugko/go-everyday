package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	n := flag.Int("n", 20, "nombre d'entrées à afficher")
	flag.Parse()

	root := "."
	if flag.NArg() > 0 {
		root = flag.Arg(0)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dush:", err)
		os.Exit(1)
	}
	info, err := os.Lstat(abs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dush:", err)
		os.Exit(1)
	}
	// Cas tordu : on a pointé dush sur un fichier. On le crache et on sort.
	if !info.IsDir() {
		fmt.Printf("%10s  %s\n", humanSize(info.Size()), abs)
		return
	}

	// +1 parce que la racine va atterrir dans le heap (elle est forcément
	// la plus grosse), et on la filtrera à l'affichage.
	t := &topN{cap: *n + 1}
	sem := make(chan struct{}, runtime.NumCPU()*2)
	walk(abs, t, sem)

	render(t, abs, *n, os.Stdout)
}
