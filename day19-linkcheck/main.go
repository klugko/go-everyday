// linkcheck — détecte les liens morts dans un Markdown ou tout un dossier de docs.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	timeout := flag.Duration("timeout", 10*time.Second, "délai max par lien externe")
	workers := flag.Int("n", 8, "liens vérifiés en parallèle")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "linkcheck — détecte les liens morts dans un Markdown ou un dossier de docs.")
		fmt.Fprintln(os.Stderr, "usage : linkcheck [-timeout 10s] [-n 8] [chemin]")
		fmt.Fprintln(os.Stderr, "le chemin est un fichier .md ou un dossier (parcouru en récursif). défaut : le dossier courant.")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		root = "."
	}

	files, err := collect(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "linkcheck :", err)
		os.Exit(1)
	}

	var all []link
	for _, f := range files {
		ls, err := links(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "linkcheck :", err)
			os.Exit(1)
		}
		all = append(all, ls...)
	}

	client := &http.Client{Timeout: *timeout}
	dead := check(all, client, *workers)

	for _, d := range dead {
		fmt.Printf("%s:%d  %s  → %s\n", d.link.file, d.link.line, d.link.url, d.reason)
	}
	if len(dead) > 0 {
		fmt.Fprintf(os.Stderr, "%d lien(s) mort(s) sur %d vérifié(s)\n", len(dead), len(all))
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "%d lien(s), tous valides\n", len(all))
}

type link struct {
	file string
	line int
	url  string
}

type result struct {
	link   link
	reason string
}

// collect renvoie les .md à examiner : le fichier seul s'il est passé tel quel,
// sinon tous ceux du dossier en descendant l'arborescence.
func collect(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var files []string
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".md") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

// On capture ce qui est entre ]( et ) : le ![](...) des images passe par le même
// motif. Un titre optionnel peut suivre l'URL, on le retire ensuite.
var linkRE = regexp.MustCompile(`\]\(([^)]+)\)`)

// links relève tous les liens d'un fichier, avec leur ligne pour pouvoir pointer
// dessus. Les blocs de code clôturés sont sautés : un lien dans un exemple n'est
// pas un lien à suivre.
func links(file string) ([]link, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var out []link
	inCode := false
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		for _, m := range linkRE.FindAllStringSubmatch(line, -1) {
			url := strings.TrimSpace(m[1])
			// [texte](url "titre") : on ne garde que l'URL, le titre dégage.
			if sp := strings.IndexAny(url, " \t"); sp >= 0 {
				url = url[:sp]
			}
			if url != "" {
				out = append(out, link{file, i + 1, url})
			}
		}
	}
	return out, nil
}

// check vérifie chaque lien et renvoie les morts, triés par fichier puis ligne.
// Tout part sur un pool de goroutines : c'est le réseau qui commande, autant
// attendre plusieurs réponses à la fois.
func check(links []link, client *http.Client, workers int) []result {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan link)
	var (
		mu   sync.Mutex
		dead []result
		wg   sync.WaitGroup
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for l := range jobs {
				if reason := checkOne(client, l); reason != "" {
					mu.Lock()
					dead = append(dead, result{l, reason})
					mu.Unlock()
				}
			}
		}()
	}
	for _, l := range links {
		jobs <- l
	}
	close(jobs)
	wg.Wait()

	sort.Slice(dead, func(i, j int) bool {
		if dead[i].link.file != dead[j].link.file {
			return dead[i].link.file < dead[j].link.file
		}
		return dead[i].link.line < dead[j].link.line
	})
	return dead
}

// schemeRE repère un lien à schéma (mailto:, tel:, ftp:…) qu'on ne sait pas
// — ou ne veut pas — vérifier. http(s) est traité à part, avant ce filet.
var schemeRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// checkOne renvoie pourquoi un lien est mort, ou "" s'il tient. Trois familles :
// les ancres et schémas inconnus sont laissés tranquilles, les chemins locaux
// sont cherchés sur le disque, le reste part sur le réseau.
func checkOne(client *http.Client, l link) string {
	url := l.url
	switch {
	case strings.HasPrefix(url, "#"):
		return ""
	case strings.HasPrefix(url, "http://"), strings.HasPrefix(url, "https://"):
		return checkHTTP(client, url)
	case schemeRE.MatchString(url):
		return "" 
	default:
		return checkLocal(l.file, url)
	}
}

// checkLocal cherche la cible sur le disque, relative au fichier qui la cite. On
// laisse tomber l'ancre (#section) et la query (?v=2) : seul le fichier pointé
// dit si le lien retombe quelque part.
func checkLocal(from, url string) string {
	target := url
	if i := strings.IndexAny(target, "#?"); i >= 0 {
		target = target[:i]
	}
	if target == "" {
		return "" 
	}
	target = filepath.Join(filepath.Dir(from), filepath.FromSlash(target))
	if _, err := os.Stat(target); err != nil {
		return "fichier introuvable"
	}
	return ""
}

// checkHTTP tape l'URL et juge sur le code de retour. On tente d'abord un HEAD,
// plus léger ; pas mal de serveurs le boudent (403, 405…), on repasse alors en
// GET avant de conclure.
func checkHTTP(client *http.Client, url string) string {
	resp, err := client.Head(url)
	if err != nil || resp.StatusCode >= 400 {
		if resp != nil {
			resp.Body.Close()
		}
		resp, err = client.Get(url)
	}
	if err != nil {
		return "injoignable"
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return ""
}
