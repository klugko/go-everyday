package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// rules : les transformations à enchaîner sur chaque nom de fichier.
// L'ordre d'application est fixe — find → case → num — parce que c'est
// celui qui a du sens : on nettoie, on normalise la casse, puis on
// numérote ce qui en sort.
type rules struct {
	find             *regexp.Regexp
	replace          string
	caseMode         string // "" | lower | upper | title
	numTmpl          string // gabarit avec {n} et {name}, "" si pas de numérotation
	start, step, pad int
}

type entry struct {
	oldPath, newPath string
	oldName, newName string
}

// plan : tout ce qu'on sait avant de toucher au disque. On le calcule en
// entier, on le valide, et seulement après on applique. C'est ce qui rend
// l'aperçu possible et le renommage sûr.
type plan struct {
	entries   []entry // uniquement les fichiers dont le nom change
	unchanged int
	conflicts []string
}

// transform applique la chaîne de règles à un nom et renvoie le nouveau.
// On découpe nom/extension et on ne touche QUE le radical : personne ne
// veut voir "photo.JPG" devenir "photo.jpg" en passant en minuscules, ni
// que la numérotation s'insère après l'extension.
func (r rules) transform(name string, i, count int) string {
	ext := filepath.Ext(name)
	stem := name[:len(name)-len(ext)]

	if r.find != nil {
		stem = r.find.ReplaceAllString(stem, r.replace)
	}
	switch r.caseMode {
	case "lower":
		stem = strings.ToLower(stem)
	case "upper":
		stem = strings.ToUpper(stem)
	case "title":
		stem = titleCase(stem)
	}
	if r.numTmpl != "" {
		n := r.start + i*r.step
		pad := r.pad
		// Pad auto : on cale la largeur sur le plus grand numéro du lot,
		// pour que tout s'aligne et se trie naturellement (001..120).
		if pad == 0 {
			max := r.start + (count-1)*r.step
			pad = len(strconv.Itoa(max))
		}
		num := fmt.Sprintf("%0*d", pad, n)
		stem = strings.ReplaceAll(r.numTmpl, "{name}", stem)
		stem = strings.ReplaceAll(stem, "{n}", num)
	}
	return stem + ext
}

// titleCase : majuscule en début de chaque mot, le reste en minuscules.
// Un "mot" commence après tout ce qui n'est pas une lettre — espace,
// underscore, tiret, chiffre. "my_file2name" → "My_File2Name".
func titleCase(s string) string {
	var b strings.Builder
	boundary := true
	for _, r := range s {
		if unicode.IsLetter(r) {
			if boundary {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			boundary = false
		} else {
			b.WriteRune(r)
			boundary = true
		}
	}
	return b.String()
}

// gather rassemble les fichiers à traiter. Un argument peut être un
// fichier (pris tel quel) ou un dossier (ses enfants directs, sans
// récursion : un renommage en masse se raisonne dossier par dossier).
// PowerShell ne développe pas les jokers pour un exe natif, d'où -glob.
func gather(paths []string, glob string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	seen := map[string]bool{}
	var files []string
	add := func(full, base string) error {
		if glob != "" {
			ok, err := filepath.Match(glob, base)
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
		if !seen[full] {
			seen[full] = true
			files = append(files, full)
		}
		return nil
	}
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			return nil, err
		}
		// Symlinks ignorés : renommer la cible via un alias est piégeux.
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			ents, err := os.ReadDir(p)
			if err != nil {
				return nil, err
			}
			for _, e := range ents {
				if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
					continue
				}
				if err := add(filepath.Join(p, e.Name()), e.Name()); err != nil {
					return nil, err
				}
			}
		} else if err := add(p, filepath.Base(p)); err != nil {
			return nil, err
		}
	}
	// Ordre stable : la numérotation suit l'ordre alphabétique des chemins.
	sort.Strings(files)
	return files, nil
}

// buildPlan calcule les renommages et détecte tout ce qui clocherait avant
// d'agir. Trois pièges à attraper : nom vide, deux sources qui visent la
// même cible, et une cible qui existe déjà sans faire partie du lot.
func buildPlan(files []string, r rules) plan {
	var p plan
	moving := map[string]bool{}
	for i, old := range files {
		base := filepath.Base(old)
		nb := r.transform(base, i, len(files))
		if nb == base {
			p.unchanged++
			continue
		}
		if nb == "" {
			p.conflicts = append(p.conflicts, fmt.Sprintf("%q deviendrait un nom vide", base))
			continue
		}
		np := filepath.Join(filepath.Dir(old), nb)
		p.entries = append(p.entries, entry{old, np, base, nb})
		moving[old] = true
	}

	// Deux fichiers qui atterrissent au même endroit : l'un écraserait
	// l'autre. On refuse plutôt que de perdre des données en silence.
	targets := map[string][]string{}
	for _, e := range p.entries {
		targets[e.newPath] = append(targets[e.newPath], e.oldName)
	}
	for tgt, srcs := range targets {
		if len(srcs) > 1 {
			sort.Strings(srcs)
			p.conflicts = append(p.conflicts, fmt.Sprintf(
				"%q est visé par %d fichiers : %s",
				filepath.Base(tgt), len(srcs), strings.Join(srcs, ", ")))
		}
	}

	// Cible déjà sur le disque ET qui ne bouge pas elle-même. Si elle
	// bouge aussi (échange a↔b), l'application en deux temps gère le cas.
	for _, e := range p.entries {
		if moving[e.newPath] {
			continue
		}
		dst, err := os.Lstat(e.newPath)
		if err != nil {
			continue
		}
		// Renommage qui ne change que la casse sur un système de fichiers
		// insensible à la casse (Windows) : la "cible" est le fichier
		// source lui-même, pas un conflit.
		if src, err := os.Lstat(e.oldPath); err == nil && os.SameFile(dst, src) {
			continue
		}
		p.conflicts = append(p.conflicts, fmt.Sprintf(
			"%q existe déjà (source : %s)", e.newName, e.oldName))
	}

	sort.Strings(p.conflicts)
	return p
}

// render écrit l'aperçu : la table old → new, le bilan, et les conflits.
func render(p plan, w io.Writer, apply bool) {
	if len(p.entries) == 0 && len(p.conflicts) == 0 {
		fmt.Fprintln(w, "Rien à renommer.")
		return
	}
	width := 0
	for _, e := range p.entries {
		if n := len(e.oldName); n > width {
			width = n
		}
	}
	for _, e := range p.entries {
		fmt.Fprintf(w, "  %-*s  →  %s\n", width, e.oldName, e.newName)
	}
	fmt.Fprintf(w, "\n%d à renommer", len(p.entries))
	if p.unchanged > 0 {
		fmt.Fprintf(w, ", %d inchangés", p.unchanged)
	}
	fmt.Fprintln(w, ".")

	if len(p.conflicts) > 0 {
		fmt.Fprintf(w, "\n%d conflit(s) — rien ne sera renommé :\n", len(p.conflicts))
		for _, c := range p.conflicts {
			fmt.Fprintf(w, "  ! %s\n", c)
		}
		return
	}
	if !apply {
		fmt.Fprintln(w, "\nAperçu seulement. Relance avec -apply pour renommer.")
	}
}

// applyPlan renomme en deux temps : tout vers des noms temporaires, puis
// les temporaires vers leurs noms finaux. Ce détour paraît superflu mais
// il rend les échanges (a↔b) et les cycles (a→b→c→a) possibles, là où un
// renommage direct écraserait un fichier au milieu de la chaîne.
func applyPlan(p plan, w io.Writer) error {
	temps := make([]string, len(p.entries))
	for i, e := range p.entries {
		tmp := e.oldPath + ".renamer-tmp-" + strconv.Itoa(i)
		if err := os.Rename(e.oldPath, tmp); err != nil {
			// Échec en phase 1 : on remet en place ce qui a déjà bougé.
			for j := i - 1; j >= 0; j-- {
				os.Rename(temps[j], p.entries[j].oldPath)
			}
			return fmt.Errorf("%s : %w", e.oldName, err)
		}
		temps[i] = tmp
	}
	var failed int
	for i, e := range p.entries {
		if err := os.Rename(temps[i], e.newPath); err != nil {
			fmt.Fprintf(w, "  ! %s : %v\n", e.newName, err)
			failed++
			continue
		}
		fmt.Fprintf(w, "  %s  →  %s\n", e.oldName, e.newName)
	}
	if failed > 0 {
		return fmt.Errorf("%d renommage(s) ont échoué", failed)
	}
	return nil
}
