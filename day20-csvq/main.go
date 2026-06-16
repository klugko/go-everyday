// csvq — exécute un sous-ensemble de SQL (SELECT … WHERE … GROUP BY …) sur un CSV.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	delim := flag.String("d", ",", "délimiteur de champ")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "csvq — du SQL sur un fichier CSV.")
		fmt.Fprintln(os.Stderr, `usage : csvq [-d ,] "SELECT cols FROM fichier.csv [WHERE …] [GROUP BY …] [ORDER BY …] [LIMIT n]"`)
		fmt.Fprintln(os.Stderr, "le fichier nommé après FROM est lu (ou '-' pour l'entrée standard). résultat en CSV sur la sortie.")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	q, err := parse(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "csvq :", err)
		os.Exit(1)
	}

	sep := []rune(*delim)[0]
	header, rows, err := readCSV(q.from, sep)
	if err != nil {
		fmt.Fprintln(os.Stderr, "csvq :", err)
		os.Exit(1)
	}

	outHeader, outRows, err := run(q, header, rows)
	if err != nil {
		fmt.Fprintln(os.Stderr, "csvq :", err)
		os.Exit(1)
	}

	w := csv.NewWriter(os.Stdout)
	w.Comma = sep
	w.Write(outHeader)
	w.WriteAll(outRows)
	w.Flush()
	if err := w.Error(); err != nil {
		fmt.Fprintln(os.Stderr, "csvq :", err)
		os.Exit(1)
	}
}

// --- modèle de requête ------------------------------------------------------

type query struct {
	cols    []selector
	from    string
	where   [][]cond 
	groupBy []string
	orderBy []orderKey
	limit   int 
}

// selector est une colonne du SELECT : soit brute, soit enveloppée dans un
// agrégat (count, sum…), avec un alias optionnel pour l'en-tête de sortie.
type selector struct {
	fn    string 
	col   string 
	alias string
}

func (s selector) label() string {
	switch {
	case s.alias != "":
		return s.alias
	case s.fn != "":
		return s.fn + "(" + s.col + ")"
	default:
		return s.col
	}
}

type cond struct {
	col string
	op  string 
	val string
	re  *regexp.Regexp 
}

type orderKey struct {
	col  string
	desc bool
}

//  lecture du CSV 

func readCSV(path string, sep rune) (header []string, rows [][]string, err error) {
	src := os.Stdin
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		src = f
	}

	r := csv.NewReader(src)
	r.Comma = sep
	r.FieldsPerRecord = -1 

	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("fichier vide")
	}
	header = all[0]
	// Un CSV exporté depuis Excel traîne souvent un BOM sur la 1re cellule.
	header[0] = strings.TrimPrefix(header[0], "\ufeff")
	return header, all[1:], nil
}

// --- exécution 
func run(q *query, header []string, rows [][]string) ([]string, [][]string, error) {
	idx := makeIndex(header)
	if err := validate(q, idx); err != nil {
		return nil, nil, err
	}
	if err := compileLikes(q); err != nil {
		return nil, nil, err
	}

	// 1. WHERE : on ne garde que les lignes qui passent le filtre.
	var kept [][]string
	for _, row := range rows {
		if matchWhere(q.where, row, idx) {
			kept = append(kept, row)
		}
	}

	// 2. agrégation : déclenchée par un GROUP BY ou la moindre fonction.
	agg := len(q.groupBy) > 0
	for _, s := range q.cols {
		agg = agg || s.fn != ""
	}

	var outHeader []string
	var outRows [][]string
	if agg {
		outHeader, outRows = aggregate(q, header, kept, idx)
	} else {
		outHeader = projectHeader(q.cols, header)
		for _, row := range kept {
			outRows = append(outRows, projectRow(q.cols, row, idx))
		}
	}

	// 3. ORDER BY puis LIMIT, sur les lignes déjà projetées.
	if err := orderRows(q.orderBy, outHeader, outRows); err != nil {
		return nil, nil, err
	}
	if q.limit >= 0 && len(outRows) > q.limit {
		outRows = outRows[:q.limit]
	}
	return outHeader, outRows, nil
}

func makeIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		// Insensible à la casse : on retient la première occurrence d'un nom.
		key := strings.ToLower(h)
		if _, seen := idx[key]; !seen {
			idx[key] = i
		}
	}
	return idx
}

// ci résout un nom de colonne déjà validé vers son indice.
func ci(idx map[string]int, name string) int { return idx[strings.ToLower(name)] }

// validate vérifie que toute colonne citée existe, avant d'entrer dans les
// boucles chaudes où une erreur serait pénible à remonter ligne par ligne.
func validate(q *query, idx map[string]int) error {
	known := func(name string) error {
		if name == "*" {
			return nil
		}
		if _, ok := idx[strings.ToLower(name)]; !ok {
			return fmt.Errorf("colonne inconnue : %s", name)
		}
		return nil
	}
	for _, s := range q.cols {
		if err := known(s.col); err != nil {
			return err
		}
	}
	for _, group := range q.where {
		for _, c := range group {
			if err := known(c.col); err != nil {
				return err
			}
		}
	}
	for _, c := range q.groupBy {
		if err := known(c); err != nil {
			return err
		}
	}
	return nil
}

func compileLikes(q *query) error {
	for gi := range q.where {
		for ci := range q.where[gi] {
			c := &q.where[gi][ci]
			if c.op != "LIKE" {
				continue
			}
			re, err := regexp.Compile(likeToRegex(c.val))
			if err != nil {
				return err
			}
			c.re = re
		}
	}
	return nil
}

// matchWhere : vrai si la ligne satisfait au moins un groupe (OU), chaque
// groupe étant un ET de conditions. Pas de clause = tout passe.
func matchWhere(where [][]cond, row []string, idx map[string]int) bool {
	if len(where) == 0 {
		return true
	}
	for _, group := range where {
		all := true
		for _, c := range group {
			if !evalCond(c, cell(row, ci(idx, c.col))) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func evalCond(c cond, val string) bool {
	switch c.op {
	case "LIKE":
		return c.re.MatchString(val)
	case "=":
		return equal(val, c.val)
	case "!=":
		return !equal(val, c.val)
	default:
		cmp := compareVals(val, c.val)
		switch c.op {
		case "<":
			return cmp < 0
		case "<=":
			return cmp <= 0
		case ">":
			return cmp > 0
		case ">=":
			return cmp >= 0
		}
	}
	return false
}

// --- projection sans agrégat 

func projectHeader(cols []selector, header []string) []string {
	var out []string
	for _, s := range cols {
		if s.fn == "" && s.col == "*" {
			out = append(out, header...)
		} else {
			out = append(out, s.label())
		}
	}
	return out
}

func projectRow(cols []selector, row []string, idx map[string]int) []string {
	var out []string
	for _, s := range cols {
		if s.fn == "" && s.col == "*" {
			out = append(out, row...)
		} else {
			out = append(out, cell(row, ci(idx, s.col)))
		}
	}
	return out
}

// --- agrégation

func aggregate(q *query, header []string, rows [][]string, idx map[string]int) ([]string, [][]string) {
	// On regroupe en gardant l'ordre de première apparition des clés.
	var keys []string
	groups := map[string][][]string{}
	for _, row := range rows {
		k := groupKey(q.groupBy, row, idx)
		if _, seen := groups[k]; !seen {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], row)
	}
	// Sans GROUP BY, un agrégat sur zéro ligne reste une ligne (count = 0).
	if len(q.groupBy) == 0 && len(keys) == 0 {
		keys = []string{""}
	}

	out := make([][]string, 0, len(keys))
	for _, k := range keys {
		g := groups[k]
		var row []string
		for _, s := range q.cols {
			switch {
			case s.fn != "":
				row = append(row, computeAgg(s.fn, s.col, g, idx))
			case s.col == "*":
				row = append(row, first(g)...)
			default:
				// Colonne brute (typiquement celle du GROUP BY) : valeur du 1er.
				if len(g) > 0 {
					row = append(row, cell(g[0], ci(idx, s.col)))
				} else {
					row = append(row, "")
				}
			}
		}
		out = append(out, row)
	}
	return projectHeader(q.cols, header), out
}

func groupKey(cols []string, row []string, idx map[string]int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = cell(row, ci(idx, c))
	}
	return strings.Join(parts, "\x00") 
}

func computeAgg(fn, col string, group [][]string, idx map[string]int) string {
	if fn == "count" {
		if col == "*" {
			return strconv.Itoa(len(group))
		}
		n := 0
		for _, row := range group {
			if strings.TrimSpace(cell(row, ci(idx, col))) != "" {
				n++
			}
		}
		return strconv.Itoa(n)
	}

	// sum/avg/min/max : numériques, on saute les cellules vides.
	var nums []float64
	for _, row := range group {
		c := strings.TrimSpace(cell(row, ci(idx, col)))
		if c == "" {
			continue
		}
		f, err := strconv.ParseFloat(c, 64)
		if err != nil {
			// Une colonne supposée chiffrée mais qui ne l'est pas : on le dit.
			return "NaN"
		}
		nums = append(nums, f)
	}
	if len(nums) == 0 {
		return ""
	}
	switch fn {
	case "sum":
		return fmtNum(sum(nums))
	case "avg":
		return fmtNum(sum(nums) / float64(len(nums)))
	case "min":
		m := nums[0]
		for _, v := range nums[1:] {
			if v < m {
				m = v
			}
		}
		return fmtNum(m)
	case "max":
		m := nums[0]
		for _, v := range nums[1:] {
			if v > m {
				m = v
			}
		}
		return fmtNum(m)
	}
	return ""
}

// --- ORDER BY

func orderRows(keys []orderKey, header []string, rows [][]string) error {
	if len(keys) == 0 {
		return nil
	}
	cols := make([]int, len(keys))
	for i, k := range keys {
		j := headerIndex(header, k.col)
		if j < 0 {
			return fmt.Errorf("colonne ORDER BY inconnue : %s (utilisez un alias)", k.col)
		}
		cols[i] = j
	}
	sort.SliceStable(rows, func(a, b int) bool {
		for i, k := range keys {
			cmp := compareVals(cell(rows[a], cols[i]), cell(rows[b], cols[i]))
			if cmp != 0 {
				if k.desc {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})
	return nil
}

// --- petits utilitaires 

func cell(row []string, i int) string {
	if i >= 0 && i < len(row) {
		return row[i]
	}
	return ""
}

func first(group [][]string) []string {
	if len(group) > 0 {
		return group[0]
	}
	return nil
}

func headerIndex(header []string, name string) int {
	for i, h := range header {
		if strings.EqualFold(h, name) {
			return i
		}
	}
	return -1
}

func sum(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

func fmtNum(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

// equal et compareVals comparent en nombres quand les deux côtés sont chiffrés,
// sinon en texte. C'est ce qu'attend l'œil : age > 9 ne doit pas dire "10 < 9".
func equal(a, b string) bool {
	if x, y, ok := bothNum(a, b); ok {
		return x == y
	}
	return a == b
}

func compareVals(a, b string) int {
	if x, y, ok := bothNum(a, b); ok {
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(a, b)
}

func bothNum(a, b string) (float64, float64, bool) {
	x, err1 := strconv.ParseFloat(strings.TrimSpace(a), 64)
	y, err2 := strconv.ParseFloat(strings.TrimSpace(b), 64)
	return x, y, err1 == nil && err2 == nil
}

// likeToRegex traduit un motif SQL (% = n'importe quoi, _ = un caractère) en
// expression régulière ancrée. Insensible à la casse, plus commode en ad hoc.
func likeToRegex(pat string) string {
	var sb strings.Builder
	sb.WriteString("(?i)^")
	for _, r := range pat {
		switch r {
		case '%':
			sb.WriteString(".*")
		case '_':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")
	return sb.String()
}
