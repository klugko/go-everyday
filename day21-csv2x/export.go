package main

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// writeJSON sort un tableau d'objets, une ligne CSV = un objet. On écrit le
// JSON à la main (et pas via une map) pour garder l'ordre des colonnes : une
// map Go se sérialise par clé triée, ce qui mélangerait l'en-tête.
func writeJSON(w io.Writer, header []string, rows [][]string) error {
	bw := bufio.NewWriter(w)
	bw.WriteString("[\n")
	for ri, row := range rows {
		bw.WriteString("  {")
		for ci, key := range header {
			if ci > 0 {
				bw.WriteString(", ")
			}
			k, _ := json.Marshal(key)
			bw.Write(k)
			bw.WriteString(": ")
			bw.WriteString(jsonValue(cell(row, ci)))
		}
		bw.WriteString("}")
		if ri < len(rows)-1 {
			bw.WriteString(",")
		}
		bw.WriteByte('\n')
	}
	bw.WriteString("]\n")
	return bw.Flush()
}

// jsonValue rend une cellule sous sa forme JSON la plus naturelle : null si
// vide, nombre ou booléen si elle en a la tête, chaîne sinon. C'est tout
// l'intérêt de passer en JSON plutôt que de laisser du texte partout.
func jsonValue(s string) string {
	switch {
	case s == "":
		return "null"
	case s == "true" || s == "false":
		return s
	case looksNumeric(s):
		return s
	default:
		b, _ := json.Marshal(s)
		return string(b)
	}
}

// looksNumeric : la cellule est-elle un nombre qu'on peut écrire tel quel en
// JSON ? On exige un ParseFloat propre, mais on refuse les zéros de tête
// (« 007 », un code postal) et le '+' : ce sont des identifiants, pas des
// quantités — ils doivent rester du texte.
func looksNumeric(s string) bool {
	if s == "" || s[0] == '+' {
		return false
	}
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return false
	}
	digits := strings.TrimPrefix(s, "-")
	if len(digits) > 1 && digits[0] == '0' && digits[1] != '.' {
		return false
	}
	return true
}

// writeMarkdown sort une table GitHub-flavored. Les pipes sont échappés et les
// sauts de ligne aplatis, sinon une cellule casserait l'alignement du tableau.
func writeMarkdown(w io.Writer, header []string, rows [][]string) error {
	bw := bufio.NewWriter(w)

	writeRow := func(cells []string) {
		bw.WriteString("|")
		for _, c := range cells {
			bw.WriteString(" ")
			bw.WriteString(mdEscape(c))
			bw.WriteString(" |")
		}
		bw.WriteByte('\n')
	}

	writeRow(header)

	// Ligne de séparation : un tiret par colonne.
	bw.WriteString("|")
	for range header {
		bw.WriteString(" --- |")
	}
	bw.WriteByte('\n')

	for _, row := range rows {
		// On normalise la largeur sur l'en-tête pour ne pas décaler le tableau.
		cells := make([]string, len(header))
		for i := range header {
			cells[i] = cell(row, i)
		}
		writeRow(cells)
	}
	return bw.Flush()
}

func mdEscape(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}
