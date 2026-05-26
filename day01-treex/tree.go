package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type node struct {
	name     string
	isDir    bool
	size     int64 
	children []*node
}

type options struct {
	maxDepth     int
	showHidden   bool
	useGitignore bool
}

// build descend récursivement et retourne le nœud + sa taille agrégée.
// récursion : chaque niveau ajoute
func build(path string, rules []rule, opts options, depth int) *node {
	info, err := os.Lstat(path)
	if err != nil {
		return nil
	}
	n := &node{name: filepath.Base(path), isDir: info.IsDir()}

	if !info.IsDir() {
		n.size = info.Size()
		return n
	}

	if opts.useGitignore {
		rules = append(rules, readGitignore(path)...)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return n
	}
	// Dossiers d'abord, alphabétique insensible à la casse.
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir() != b.IsDir() {
			return a.IsDir()
		}
		return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
	})

	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue 
		}
		if !opts.showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		child := filepath.Join(path, name)
		if opts.useGitignore && isIgnored(rules, child, e.IsDir()) {
			continue
		}
		if opts.maxDepth > 0 && depth+1 > opts.maxDepth {
			continue
		}
		if c := build(child, rules, opts, depth+1); c != nil {
			n.children = append(n.children, c)
			n.size += c.size
		}
	}
	return n
}

// humanSize formate  (KiB, MiB, ...).
func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for x := b / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

//  affiche l'arbre avec les caractères box-drawing.
func render(w io.Writer, n *node, prefix string, isLast, isRoot bool) {
	if isRoot {
		fmt.Fprintf(w, "%s  [%s]\n", n.name, humanSize(n.size))
	} else {
		marker := "├── "
		if isLast {
			marker = "└── "
		}
		fmt.Fprintf(w, "%s%s%s  [%s]\n", prefix, marker, n.name, humanSize(n.size))
	}
	for i, c := range n.children {
		next := prefix
		if !isRoot {
			if isLast {
				next += "    "
			} else {
				next += "│   "
			}
		}
		render(w, c, next, i == len(n.children)-1, false)
	}
}
