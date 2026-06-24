// license — insère un en-tête de licence en tête de chaque fichier source d'un
// projet. Idempotent : relancé, il ne double jamais l'en-tête.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	lic := flag.String("license", "mit", "licence : mit | apache-2.0 | gpl-3.0 | bsd-3")
	name := flag.String("name", "", "détenteur du copyright (obligatoire sauf avec -text)")
	year := flag.Int("year", time.Now().Year(), "année du copyright")
	textFile := flag.String("text", "", "fichier d'en-tête personnalisé (remplace -license/-name)")
	dry := flag.Bool("n", false, "montre ce qui changerait sans rien écrire")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "license — insère un en-tête de licence dans les fichiers source.")
		fmt.Fprintln(os.Stderr, "usage : license -name \"Alice\" [-license mit] [-year 2026] [-n] [chemin]")
		fmt.Fprintln(os.Stderr, "       license -text MON_ENTETE.txt [chemin]")
		flag.PrintDefaults()
	}
	flag.Parse()

	root := flag.Arg(0)
	if root == "" {
		root = "."
	}

	lines, marker, err := buildHeader(*lic, *name, *year, *textFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "license :", err)
		os.Exit(2)
	}

	changed, err := run(root, lines, marker, *dry)
	if err != nil {
		fmt.Fprintln(os.Stderr, "license :", err)
		os.Exit(1)
	}
	verb := "modifié(s)"
	if *dry {
		verb = "à modifier"
	}
	fmt.Fprintf(os.Stderr, "%d fichier(s) %s\n", changed, verb)
}

// buildHeader renvoie le texte brut de l'en-tête (avant mise en commentaire) et
// le marqueur qui sert à détecter sa présence. Avec -text, l'utilisateur fournit
// tout ; sinon on fabrique un en-tête SPDX standard.
func buildHeader(lic, name string, year int, textFile string) (lines []string, marker string, err error) {
	if textFile != "" {
		data, err := os.ReadFile(textFile)
		if err != nil {
			return nil, "", err
		}
		lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		// Sans SPDX dans un en-tête maison, on se rabat sur sa première ligne
		// comme sentinelle d'idempotence.
		marker = spdxMarker
		if !containsMarker(lines, spdxMarker) {
			marker = lines[0]
		}
		return lines, marker, nil
	}

	id, ok := spdxID[strings.ToLower(lic)]
	if !ok {
		return nil, "", fmt.Errorf("licence inconnue %q (mit, apache-2.0, gpl-3.0, bsd-3)", lic)
	}
	if strings.TrimSpace(name) == "" {
		return nil, "", fmt.Errorf("-name est requis (le détenteur du copyright)")
	}
	lines = []string{
		"Copyright (c) " + strconv.Itoa(year) + " " + name,
		spdxMarker + ": " + id,
	}
	return lines, spdxMarker, nil
}

func containsMarker(lines []string, marker string) bool {
	for _, l := range lines {
		if strings.Contains(l, marker) {
			return true
		}
	}
	return false
}

// run parcourt l'arbre et traite chaque fichier dont on connaît le style de
// commentaire. Retourne le nombre de fichiers modifiés (ou à modifier en -n).
func run(root string, headerLines []string, marker string, dry bool) (int, error) {
	changed := 0
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		prefix, ok := commentPrefix[strings.ToLower(filepath.Ext(p))]
		if !ok {
			return nil // langage non géré : on ne touche pas
		}
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "license : %s ignoré (%v)\n", p, err)
			return nil
		}
		header := render(prefix, headerLines)
		out, did := apply(string(data), header, marker)
		if !did {
			return nil
		}
		changed++
		if dry {
			fmt.Printf("[dry] %s\n", p)
			return nil
		}
		// On garde les permissions d'origine plutôt que d'imposer 0644.
		info, _ := d.Info()
		mode := os.FileMode(0644)
		if info != nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(p, []byte(out), mode); err != nil {
			return err
		}
		fmt.Printf("%s\n", p)
		return nil
	})
	return changed, err
}

var skipped = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true,
}

func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipped[name]
}
