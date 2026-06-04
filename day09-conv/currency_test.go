package main

import "testing"

// Une réponse Frankfurter figée : on teste le décodage et l'application du
// taux sans toucher au réseau.
const sample = `{"amount":1.0,"base":"USD","date":"2026-06-04","rates":{"EUR":0.921}}`

func TestParseAndApplyRate(t *testing.T) {
	r, err := parseRates([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if r.Date != "2026-06-04" || r.Base != "USD" {
		t.Fatalf("entête mal lu : %+v", r)
	}

	got, err := applyRate(50, "EUR", r)
	if err != nil {
		t.Fatal(err)
	}
	if want := 46.05; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("50 USD = %g EUR, attendu %g", got, want)
	}
}

func TestApplyRateUnknown(t *testing.T) {
	r, _ := parseRates([]byte(sample))
	if _, err := applyRate(50, "JPY", r); err == nil {
		t.Error("une devise absente des taux devrait être une erreur")
	}
}
