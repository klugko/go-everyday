package main

import (
	"reflect"
	"testing"
)

func TestKeys(t *testing.T) {
	in := `
# Configuration de la base
DATABASE_URL=postgres://localhost/app
export API_KEY=secret

  PORT = 8080
# ligne de commentaire
DATABASE_URL=doublon-ignore
pas une affectation
clé avec espace=non comptée
`
	got := Keys(in)
	want := []string{"DATABASE_URL", "API_KEY", "PORT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Keys = %v, attendu %v", got, want)
	}
}

func TestCompare(t *testing.T) {
	example := []string{"DATABASE_URL", "API_KEY", "PORT"}
	env := []string{"DATABASE_URL", "PORT", "DEBUG"}

	got := Compare(example, env)
	want := Diff{
		Missing: []string{"API_KEY"},
		Extra:   []string{"DEBUG"},  
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Compare = %+v, attendu %+v", got, want)
	}
	if got.OK() {
		t.Error("Compare.OK = true, attendu false avec un diff non vide")
	}
}

func TestCompareIdentique(t *testing.T) {
	keys := []string{"A", "B", "C"}
	got := Compare(keys, keys)
	if !got.OK() {
		t.Errorf("Compare.OK = false alors que les fichiers sont identiques : %+v", got)
	}
}

func TestCompareOrdrePreserve(t *testing.T) {
	// L'ordre des manquantes suit le modèle, pas l'ordre alphabétique.
	example := []string{"Z", "A", "M"}
	env := []string{}
	got := Compare(example, env)
	want := []string{"Z", "A", "M"}
	if !reflect.DeepEqual(got.Missing, want) {
		t.Errorf("Missing = %v, attendu %v (ordre du modèle)", got.Missing, want)
	}
}
