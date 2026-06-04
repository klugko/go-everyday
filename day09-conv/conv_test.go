package main

import (
	"math"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestConvertOfflineLinear(t *testing.T) {
	cases := []struct {
		value    float64
		from, to string
		want     float64
		wantSym  string
	}{
		{1, "km", "m", 1000, "m"},
		{10, "km", "mi", 6.21371192, "mi"},
		{1, "mi", "km", 1.609344, "km"},
		{80, "kg", "lb", 176.36980975, "lb"},
		{1, "lb", "g", 453.59237, "g"},
		{1, "tonne", "kg", 1000, "kg"},
		{12, "in", "cm", 30.48, "cm"},
	}
	for _, c := range cases {
		res, ok, err := ConvertOffline(c.value, c.from, c.to)
		if err != nil || !ok {
			t.Fatalf("%g %s→%s : ok=%v err=%v", c.value, c.from, c.to, ok, err)
		}
		if !almost(res.Value, c.want) {
			t.Errorf("%g %s→%s = %g, attendu %g", c.value, c.from, c.to, res.Value, c.want)
		}
		if res.ToSym != c.wantSym {
			t.Errorf("symbole %q, attendu %q", res.ToSym, c.wantSym)
		}
	}
}

func TestConvertOfflineTemperature(t *testing.T) {
	cases := []struct {
		value    float64
		from, to string
		want     float64
	}{
		{100, "c", "f", 212},
		{32, "f", "c", 0},
		{0, "c", "k", 273.15},
		{300, "k", "c", 26.85},
		{-40, "c", "f", -40},          
		{100, "°C", "fahrenheit", 212},
	}
	for _, c := range cases {
		res, ok, err := ConvertOffline(c.value, c.from, c.to)
		if err != nil || !ok {
			t.Fatalf("%g %s→%s : ok=%v err=%v", c.value, c.from, c.to, ok, err)
		}
		if !almost(res.Value, c.want) {
			t.Errorf("%g %s→%s = %g, attendu %g", c.value, c.from, c.to, res.Value, c.want)
		}
	}
}

func TestConvertOfflineMismatch(t *testing.T) {
	// km → kg : reconnu mais incohérent → ok=true, erreur.
	if _, ok, err := ConvertOffline(1, "km", "kg"); !ok || err == nil {
		t.Errorf("km→kg devrait être reconnu et refusé (ok=%v err=%v)", ok, err)
	}
	// température mélangée à autre chose.
	if _, ok, err := ConvertOffline(1, "c", "m"); !ok || err == nil {
		t.Errorf("c→m devrait être refusé (ok=%v err=%v)", ok, err)
	}
}

func TestConvertOfflineDefersToCurrency(t *testing.T) {
	// Deux inconnues physiques : on laisse la main aux devises (ok=false).
	if _, ok, err := ConvertOffline(1, "usd", "eur"); ok || err != nil {
		t.Errorf("usd→eur devrait passer aux devises (ok=%v err=%v)", ok, err)
	}
}
