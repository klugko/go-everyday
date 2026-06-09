// clip — historique de presse-papiers : retrouve ce que tu as copié tout à l'heure.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// On garde au plus les dernières copies : un presse-papiers vit longtemps en
// fond, inutile de laisser le fichier grossir sans fin.
const maxEntries = 200

// entry : une copie, avec l'heure pour répondre au « il y a 10 minutes ».
type entry struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

func main() {
	dir := flag.String("dir", defaultDir(), "dossier de l'historique")
	watch := flag.Bool("watch", false, "surveille le presse-papiers et note chaque copie")
	every := flag.Duration("every", time.Second, "intervalle de scrutation en mode --watch")
	get := flag.Int("get", 0, "recopie l'entrée N dans le presse-papiers (1 = la plus récente)")
	clear := flag.Bool("clear", false, "vide l'historique")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "clip — historique de presse-papiers.")
		fmt.Fprintln(os.Stderr, "usage : clip [--watch] [--get N] [--clear] [--dir dossier]")
		fmt.Fprintln(os.Stderr, "lance 'clip --watch' en fond, puis 'clip' pour relire l'historique.")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*dir, *watch, *every, *get, *clear); err != nil {
		fmt.Fprintln(os.Stderr, "clip :", err)
		os.Exit(1)
	}
}

func run(dir string, watch bool, every time.Duration, get int, clear bool) error {
	path := histFile(dir)

	switch {
	case clear:
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("historique vidé")
		return nil

	case watch:
		return watchLoop(path, every)

	case get > 0:
		entries, err := load(path)
		if err != nil {
			return err
		}
		e, err := pick(entries, get)
		if err != nil {
			return err
		}
		if err := writeClipboard(e.Text); err != nil {
			return err
		}
		fmt.Println("recopié :", preview(e.Text))
		return nil

	default:
		entries, err := load(path)
		if err != nil {
			return err
		}
		fmt.Print(formatList(entries, time.Now()))
		return nil
	}
}

// watchLoop scrute le presse-papiers et note chaque nouveau contenu. On part de
// ce qui est déjà collé pour ne pas l'enregistrer une fois de plus au démarrage.
func watchLoop(path string, every time.Duration) error {
	last, _ := readClipboard()
	fmt.Fprintln(os.Stderr, "clip surveille le presse-papiers (Ctrl+C pour arrêter)")
	for {
		cur, err := readClipboard()
		if err == nil && cur != "" && cur != last {
			last = cur
			if err := record(path, cur, time.Now()); err != nil {
				fmt.Fprintln(os.Stderr, "clip :", err)
			}
		}
		time.Sleep(every)
	}
}

// pick retrouve une entrée par son rang d'affichage : 1 = la plus récente.
func pick(entries []entry, n int) (entry, error) {
	if n < 1 || n > len(entries) {
		return entry{}, fmt.Errorf("pas d'entrée n°%d", n)
	}
	return entries[len(entries)-n], nil
}

// record ajoute une copie en queue d'historique. On ignore un doublon collé
// juste après le même contenu, et on rogne le plus vieux au-delà du plafond.
func record(path, text string, now time.Time) error {
	entries, err := load(path)
	if err != nil {
		return err
	}
	if n := len(entries); n > 0 && entries[n-1].Text == text {
		return nil
	}
	entries = append(entries, entry{At: now, Text: text})
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	return save(path, entries)
}

// load lit l'historique, une entrée JSON par ligne. Une copie peut contenir des
// retours à la ligne : le JSON les échappe, donc une ligne = une entrée intacte.
func load(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []entry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) 
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var e entry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}

// save réécrit l'historique d'un bloc — le plafonnement impose de tout reposer.
func save(path string, entries []entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, e := range entries {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// formatList affiche le plus récent en haut : on lit du dernier copié vers le
// plus ancien, en numérotant 1, 2, 3… dans cet ordre.
func formatList(entries []entry, now time.Time) string {
	if len(entries) == 0 {
		return "historique vide — lance 'clip --watch' pour enregistrer tes copies.\n"
	}
	var b strings.Builder
	for i := len(entries) - 1; i >= 0; i-- {
		rang := len(entries) - i
		fmt.Fprintf(&b, "%3d  %-12s  %s\n", rang, humanAge(now, entries[i].At), preview(entries[i].Text))
	}
	return b.String()
}

// humanAge dit l'âge d'une copie à la louche, comme on le dirait à l'oral.
func humanAge(now, then time.Time) string {
	d := now.Sub(then)
	switch {
	case d < time.Minute:
		return "à l'instant"
	case d < time.Hour:
		return fmt.Sprintf("il y a %d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("il y a %d h", int(d.Hours()))
	default:
		return fmt.Sprintf("il y a %d j", int(d.Hours()/24))
	}
}

// preview aplatit une copie sur une ligne et la coupe : la liste reste lisible
// même quand on a copié un pavé de texte.
func preview(text string) string {
	flat := strings.Join(strings.Fields(text), " ")
	const max = 60
	if r := []rune(flat); len(r) > max {
		return string(r[:max]) + "…"
	}
	return flat
}

func histFile(dir string) string {
	return filepath.Join(dir, "history.jsonl")
}

// defaultDir : $CLIP_DIR s'il est défini, sinon ~/.clip.
func defaultDir() string {
	if d := os.Getenv("CLIP_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "clip" 
	}
	return filepath.Join(home, ".clip")
}
