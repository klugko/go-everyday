package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type dupGroup struct {
	size  int64
	paths []string
}

// scan descend récursivement et groupe les fichiers par taille. C'est la
// première étape du dédoublonnage : deux fichiers de tailles différentes
// ne peuvent pas être identiques. Un stat coûte mille fois moins qu'un
// hash c'est un bon moyen de réduire le nombre de candidats à hasher.
func scan(root string) map[int64][]string {
	bySize := map[int64][]string{}
	var walk func(string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			// Pas de symlinks : boucle infinie possible, et dédoublonner
			// un lien n'a aucun sens de toute façon.
			if e.Type()&os.ModeSymlink != 0 {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if e.IsDir() {
				walk(p)
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			sz := info.Size()
			// Les fichiers vides sont initules
			if sz == 0 {
				continue
			}
			bySize[sz] = append(bySize[sz], p)
		}
	}
	walk(root)
	return bySize
}

// On copie en streaming, le fichier
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// renvoie les groupes de doublons. Les groupes de taille 1 sont jetés.
func group(bySize map[int64][]string, workers int) map[string]*dupGroup {
	type cand struct {
		path string
		size int64
	}
	type result struct {
		path string
		size int64
		hash string
	}

	// les fichiers seuls de leur taille ne peuvent pas être doublons
	var cands []cand
	for sz, paths := range bySize {
		if len(paths) < 2 {
			continue
		}
		for _, p := range paths {
			cands = append(cands, cand{p, sz})
		}
	}

	jobs := make(chan cand)
	out := make(chan result)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				h, err := hashFile(j.path)
				if err != nil {
					continue
				}
				out <- result{j.path, j.size, h}
			}
		}()
	}
	go func() {
		for _, c := range cands {
			jobs <- c
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(out)
	}()

	byHash := map[string]*dupGroup{}
	for r := range out {
		g, ok := byHash[r.hash]
		if !ok {
			g = &dupGroup{size: r.size}
			byHash[r.hash] = g
		}
		g.paths = append(g.paths, r.path)
	}
	for h, g := range byHash {
		if len(g.paths) < 2 {
			delete(byHash, h)
			continue
		}
		// Tri de la sortie 
		sort.Strings(g.paths)
	}
	return byHash
}

// groupes triés par espace récupérable décroissant.
// Le plus rentable à supprimer d'abord
func sortedGroups(groups map[string]*dupGroup) []*dupGroup {
	list := make([]*dupGroup, 0, len(groups))
	for _, g := range groups {
		list = append(list, g)
	}
	sort.Slice(list, func(i, j int) bool {
		wi := list[i].size * int64(len(list[i].paths)-1)
		wj := list[j].size * int64(len(list[j].paths)-1)
		if wi != wj {
			return wi > wj
		}
		return list[i].paths[0] < list[j].paths[0]
	})
	return list
}

// humanSize : KiB / MiB / GiB / ...
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

//  affiche les groupes et le total récupérable.
func render(list []*dupGroup, root string, w io.Writer) {
	if len(list) == 0 {
		fmt.Fprintln(w, "Pas de doublons.")
		return
	}
	var total int64
	for i, g := range list {
		waste := g.size * int64(len(g.paths)-1)
		total += waste
		fmt.Fprintf(w, "\n[%d] %s par fichier, %d copies — %s récupérable\n",
			i+1, humanSize(g.size), len(g.paths), humanSize(waste))
		for j, p := range g.paths {
			rel, err := filepath.Rel(root, p)
			if err != nil {
				rel = p
			}
			fmt.Fprintf(w, "  %d) %s\n", j+1, rel)
		}
	}
	fmt.Fprintf(w, "\nTotal : %s récupérable sur %d groupes.\n",
		humanSize(total), len(list))
}
