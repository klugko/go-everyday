package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"strings"
	"testing"
)

// rng renvoie un générateur à graine fixe : les tests doivent rejouer le même
// jeu à chaque exécution.
func rng() *rand.Rand { return rand.New(rand.NewSource(42)) }

func TestSeedIsReproducible(t *testing.T) {
	fields := []string{"prenom", "nom", "email"}
	var a, b bytes.Buffer
	if err := writeCSV(&a, fields, 5, rng()); err != nil {
		t.Fatal(err)
	}
	if err := writeCSV(&b, fields, 5, rng()); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Errorf("même graine, sorties différentes :\n%s\n---\n%s", a.String(), b.String())
	}
}

func TestCSVHeaderAndCount(t *testing.T) {
	fields := []string{"prenom", "ville", "cp"}
	var b bytes.Buffer
	if err := writeCSV(&b, fields, 3, rng()); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	// 1 en-tête + 3 fiches.
	if len(lines) != 4 {
		t.Fatalf("attendu 4 lignes, obtenu %d :\n%s", len(lines), b.String())
	}
	if lines[0] != "prenom,ville,cp" {
		t.Errorf("en-tête = %q", lines[0])
	}
}

func TestJSONIsValid(t *testing.T) {
	fields := []string{"prenom", "nom", "email", "tel"}
	var b bytes.Buffer
	if err := writeJSON(&b, fields, 4, rng()); err != nil {
		t.Fatal(err)
	}
	var got []map[string]string
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v\n%s", err, b.String())
	}
	if len(got) != 4 {
		t.Fatalf("attendu 4 objets, obtenu %d", len(got))
	}
	// Chaque objet porte exactement les champs demandés.
	for _, o := range got {
		for _, f := range fields {
			if _, ok := o[f]; !ok {
				t.Errorf("champ manquant %q dans %v", f, o)
			}
		}
	}
}

// L'email doit dériver du nom : minuscules, sans accent, prenom.nom au début.
func TestEmailMatchesName(t *testing.T) {
	r := rng()
	p := person{prenom: "Léa", nom: "François"}
	e := email(p, r)
	if !strings.HasPrefix(e, "lea.francois") {
		t.Errorf("email %q ne dérive pas du nom", e)
	}
	if !strings.Contains(e, "@") {
		t.Errorf("email %q sans domaine", e)
	}
}

// Ville et code postal sont tirés ensemble : ils doivent former une vraie paire.
func TestCityZipPaired(t *testing.T) {
	r := rng()
	for i := 0; i < 50; i++ {
		p := newPerson(r)
		ok := false
		for _, v := range villes {
			if v.nom == p.ville && v.cp == p.cp {
				ok = true
				break
			}
		}
		if !ok {
			t.Fatalf("paire ville/CP incohérente : %s / %s", p.ville, p.cp)
		}
	}
}

func TestPhoneFormat(t *testing.T) {
	r := rng()
	for i := 0; i < 20; i++ {
		s := phone(r)
		// "0X XX XX XX XX" : 14 caractères, démarre par 06 ou 07.
		if len(s) != 14 {
			t.Fatalf("téléphone mal formaté : %q", s)
		}
		if !strings.HasPrefix(s, "06") && !strings.HasPrefix(s, "07") {
			t.Errorf("préfixe mobile inattendu : %q", s)
		}
	}
}

func TestParseFields(t *testing.T) {
	if _, err := parseFields("prenom, email , cp"); err != nil {
		t.Errorf("liste valide refusée : %v", err)
	}
	if _, err := parseFields("prenom,xxx"); err == nil {
		t.Errorf("champ inconnu accepté")
	}
	if _, err := parseFields("  ,  "); err == nil {
		t.Errorf("liste vide acceptée")
	}
}
