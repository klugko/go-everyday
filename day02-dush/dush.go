package main

import (
	"container/heap"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
)

type entry struct {
	path string
	size int64
	dir  bool
}

// minHeap : on garde au plus `cap` éléments. Si le candidat est plus
// grand que le minimum, on remplace le minimum. Mémoire plafonnée à N.
type minHeap []entry

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].size < h[j].size }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(entry)) }
func (h *minHeap) Pop() any          { o := *h; n := len(o); v := o[n-1]; *h = o[:n-1]; return v }

type topN struct {
	mu  sync.Mutex
	h   minHeap
	cap int
}

func (t *topN) add(e entry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.h.Len() < t.cap {
		heap.Push(&t.h, e)
		return
	}
	if e.size > t.h[0].size {
		t.h[0] = e
		heap.Fix(&t.h, 0)
	}
}

// walk descend récursivement. Pour chaque sous-dossier on tente de spawn
// une goroutine ; si le sémaphore est plein, on continue inline dans le
// goroutine courant. Le compte de goroutines reste plafonné, et la
// récursion fait toujours avancer le boulot — pas de deadlock possible.
func walk(path string, t *topN, sem chan struct{}) int64 {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	var total atomic.Int64
	var wg sync.WaitGroup

	for _, e := range entries {
		sub := filepath.Join(path, e.Name())
		// Pas de symlinks : on évite les boucles infinies 
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				continue
			}
			sz := info.Size()
			t.add(entry{sub, sz, false})
			total.Add(sz)
			continue
		}
		wg.Add(1)
		select {
		case sem <- struct{}{}:
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				total.Add(walk(sub, t, sem))
			}()
		default:
			total.Add(walk(sub, t, sem))
			wg.Done()
		}
	}
	wg.Wait()

	sz := total.Load()
	t.add(entry{path, sz, true})
	return sz
}

// humanSize : formatage KiB / MiB / GiB / ...
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

// Affichage taille décroissante et affiche les n premiers,
// en sautant la racine (forcément la plus grosse, on la regarde pas).
func render(t *topN, root string, n int, w io.Writer) {
	results := make([]entry, t.h.Len())
	copy(results, t.h)
	sort.Slice(results, func(i, j int) bool { return results[i].size > results[j].size })

	shown := 0
	for _, e := range results {
		if e.path == root {
			continue
		}
		if shown >= n {
			break
		}
		rel, err := filepath.Rel(root, e.path)
		if err != nil {
			rel = e.path
		}
		if e.dir {
			rel += string(filepath.Separator)
		}
		fmt.Fprintf(w, "%10s  %s\n", humanSize(e.size), rel)
		shown++
	}
}
