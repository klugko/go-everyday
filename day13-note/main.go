// note — jette une note rapide dans le Markdown du jour, daté et horodaté.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	dir := flag.String("dir", defaultDir(), "dossier où ranger les journaux")
	show := flag.Bool("show", false, "affiche le fichier du jour au lieu d'écrire")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "note — range une note rapide dans le Markdown du jour.")
		fmt.Fprintln(os.Stderr, "usage : note [--dir dossier] [--show] [texte...]")
		fmt.Fprintln(os.Stderr, "sans texte, lit l'entrée standard.")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*dir, *show, strings.Join(flag.Args(), " "), time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "note :", err)
		os.Exit(1)
	}
}

func run(dir string, show bool, text string, now time.Time) error {
	path := noteFile(dir, now)

	if show {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(data)
		return err
	}

	// Sans argument, on récupère ce qui arrive sur stdin — pratique en pipe.
	if text == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		text = string(data)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("rien à noter")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := appendNote(path, now, text); err != nil {
		return err
	}
	fmt.Println("noté dans", path)
	return nil
}

// noteFile : un fichier par jour, nommé par sa date pour qu'un simple ls suffise.
func noteFile(dir string, now time.Time) string {
	return filepath.Join(dir, now.Format("2006-01-02")+".md")
}

// appendNote ajoute une entrée horodatée à la fin du fichier du jour. Le titre
// daté n'est écrit qu'à la première note : on le déduit d'un fichier encore vide.
func appendNote(path string, now time.Time, text string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	var b strings.Builder
	if info.Size() == 0 {
		b.WriteString(header(now))
	}
	b.WriteString(entry(now, text))
	_, err = f.WriteString(b.String())
	return err
}

// header : le titre de la journée, une seule fois en haut du fichier.
func header(now time.Time) string {
	return "# " + now.Format("2006-01-02") + "\n\n"
}

// entry : l'heure en sous-titre, puis le texte de la note.
func entry(now time.Time, text string) string {
	return fmt.Sprintf("## %s\n\n%s\n\n", now.Format("15:04"), text)
}

// defaultDir : $NOTE_DIR s'il est défini, sinon ~/notes.
func defaultDir() string {
	if d := os.Getenv("NOTE_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "notes" // dernier recours : le dossier courant
	}
	return filepath.Join(home, "notes")
}
