package main

import (
	"reflect"
	"testing"
)

// jeu de données partagé par la plupart des tests.
var (
	header = []string{"name", "city", "age", "salary"}
	rows   = [][]string{
		{"Alice", "Paris", "30", "3200"},
		{"Bob", "Lyon", "25", "2800"},
		{"Chloe", "Paris", "42", "4100"},
		{"David", "Lyon", "35", "3600"},
		{"Emma", "Paris", "28", "3000"},
	}
)

// exec parse la requête et l'exécute sur le jeu partagé, en échouant le test
// à la moindre erreur — la plupart des cas n'ont qu'un résultat à vérifier.
func exec(t *testing.T, sql string) ([]string, [][]string) {
	t.Helper()
	q, err := parse(sql)
	if err != nil {
		t.Fatalf("parse(%q) : %v", sql, err)
	}
	h, r, err := run(q, header, rows)
	if err != nil {
		t.Fatalf("run(%q) : %v", sql, err)
	}
	return h, r
}

func TestWhereAndOr(t *testing.T) {
	// (city=Paris ET age>28) OU name=Bob.
	_, got := exec(t, "SELECT name FROM x WHERE city = 'Paris' AND age > 28 OR name = 'Bob'")
	var names []string
	for _, r := range got {
		names = append(names, r[0])
	}
	want := []string{"Alice", "Bob", "Chloe"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("noms = %v, attendu %v", names, want)
	}
}

func TestWhereNumericVsString(t *testing.T) {
	// age > 9 doit comparer en nombres, pas en texte (sinon "30" < "9").
	_, got := exec(t, "SELECT name FROM x WHERE age > 9")
	if len(got) != len(rows) {
		t.Errorf("comparaison numérique cassée : %d lignes au lieu de %d", len(got), len(rows))
	}
}

func TestSelectStarKeepsAllColumns(t *testing.T) {
	h, got := exec(t, "SELECT * FROM x WHERE name = 'Bob'")
	if !reflect.DeepEqual(h, header) {
		t.Errorf("en-tête = %v, attendu %v", h, header)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0], rows[1]) {
		t.Errorf("ligne = %v, attendu %v", got, rows[1])
	}
}

func TestGroupByWithAggregates(t *testing.T) {
	h, got := exec(t, "SELECT city, count(*) AS n, sum(salary) AS masse, max(age) FROM x GROUP BY city ORDER BY city")
	wantH := []string{"city", "n", "masse", "max(age)"}
	if !reflect.DeepEqual(h, wantH) {
		t.Fatalf("en-tête = %v, attendu %v", h, wantH)
	}
	want := [][]string{
		{"Lyon", "2", "6400", "35"},
		{"Paris", "3", "10300", "42"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("résultat = %v, attendu %v", got, want)
	}
}

func TestGlobalAggregateNoGroup(t *testing.T) {
	_, got := exec(t, "SELECT count(*) AS total, avg(age) FROM x")
	want := [][]string{{"5", "32"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("résultat = %v, attendu %v", got, want)
	}
}

func TestLikeCaseInsensitive(t *testing.T) {
	_, got := exec(t, "SELECT name FROM x WHERE name LIKE '%E'") 
	var names []string
	for _, r := range got {
		names = append(names, r[0])
	}
	want := []string{"Alice", "Chloe"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("LIKE = %v, attendu %v", names, want)
	}
}

func TestOrderByDescAndLimit(t *testing.T) {
	_, got := exec(t, "SELECT name, age FROM x ORDER BY age DESC LIMIT 2")
	want := [][]string{{"Chloe", "42"}, {"David", "35"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("résultat = %v, attendu %v", got, want)
	}
}

func TestUnknownColumnIsReported(t *testing.T) {
	q, err := parse("SELECT firstname FROM x")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := run(q, header, rows); err == nil {
		t.Error("une colonne inconnue devrait être signalée")
	}
}

func TestParseErrors(t *testing.T) {
	for _, sql := range []string{
		"FROM x",                       
		"SELECT name",                  
		"SELECT name FROM x WHERE age",
	} {
		if _, err := parse(sql); err == nil {
			t.Errorf("parse(%q) aurait dû échouer", sql)
		}
	}
}

func TestTokenizeKeepsQuotedStrings(t *testing.T) {
	toks, err := tokenize("WHERE name = 'fish and chips'")
	if err != nil {
		t.Fatal(err)
	}
	last := toks[len(toks)-1]
	if !last.quoted || last.s != "fish and chips" {
		t.Errorf("chaîne quotée mal lue : %+v", last)
	}
}
