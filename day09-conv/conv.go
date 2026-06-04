package main

import (
	"fmt"
	"strings"
)

// Result porte le résultat d'une conversion et de quoi l'afficher
// proprement : la valeur, les symboles des deux côtés, et une note
// optionnelle (pour les devises, la date du taux).
type Result struct {
	Value   float64
	FromSym string
	ToSym   string
	Note    string
}

// unit décrit une unité linéaire : son facteur vers l'unité de base de sa
// dimension (le mètre pour les longueurs, le gramme pour les masses) et le
// symbole d'affichage canonique. dim sert à refuser les conversions entre
// dimensions incompatibles (convertir des km en kg n'a aucun sens).
type unit struct {
	factor float64
	sym    string
	dim    string
}

// Une seule table, des alias multiples pointant vers la même unité : on
// accepte ce que l'utilisateur tape spontanément ("mile", "miles", "mi")
// sans multiplier les branches de code.
var units = map[string]unit{
	// Longueurs — base : le mètre.
	"mm": {0.001, "mm", "longueur"},
	"cm": {0.01, "cm", "longueur"},
	"dm": {0.1, "dm", "longueur"},
	"m":  {1, "m", "longueur"},
	"km": {1000, "km", "longueur"},
	"in": {0.0254, "in", "longueur"}, "inch": {0.0254, "in", "longueur"}, "pouce": {0.0254, "in", "longueur"},
	"ft": {0.3048, "ft", "longueur"}, "pied": {0.3048, "ft", "longueur"},
	"yd": {0.9144, "yd", "longueur"},
	"mi": {1609.344, "mi", "longueur"}, "mile": {1609.344, "mi", "longueur"}, "miles": {1609.344, "mi", "longueur"},
	"nmi": {1852, "nmi", "longueur"},

	// Masses — base : le gramme.
	"mg": {0.001, "mg", "poids"},
	"g":  {1, "g", "poids"},
	"kg": {1000, "kg", "poids"},
	"t":  {1e6, "t", "poids"}, "tonne": {1e6, "t", "poids"},
	"oz": {28.349523125, "oz", "poids"}, "once": {28.349523125, "oz", "poids"},
	"lb": {453.59237, "lb", "poids"}, "livre": {453.59237, "lb", "poids"},
	"st": {6350.29318, "st", "poids"}, // stone
}

// ConvertOffline tente une conversion sans réseau : longueurs, poids,
// températures. Le booléen dit si les unités relèvent bien de l'un de ces
// domaines ; s'il est faux, l'appelant bascule sur les devises. Une erreur
// avec ok=true signale une demande reconnue mais incohérente (km vers kg).
func ConvertOffline(value float64, from, to string) (Result, bool, error) {
	from, to = normalize(from), normalize(to)

	// La température est non linéaire (décalages d'origine), donc traitée à
	// part avant la table des facteurs.
	ft, fIsTemp := tempUnit(from)
	tt, tIsTemp := tempUnit(to)
	if fIsTemp || tIsTemp {
		if !fIsTemp || !tIsTemp {
			return Result{}, true, fmt.Errorf("une température ne se convertit qu'en température")
		}
		c := toCelsius(value, ft)
		return Result{Value: fromCelsius(c, tt), FromSym: tempSym(ft), ToSym: tempSym(tt)}, true, nil
	}

	fu, fok := units[from]
	tu, tok := units[to]
	if !fok && !tok {
		return Result{}, false, nil 
	}
	if !fok || !tok {
		return Result{}, true, fmt.Errorf("unité inconnue dans le domaine physique : %q", pick(fok, to, from))
	}
	if fu.dim != tu.dim {
		return Result{}, true, fmt.Errorf("on ne convertit pas %s (%s) en %s (%s)", fu.sym, fu.dim, tu.sym, tu.dim)
	}
	// Passage par l'unité de base commune : value*fu.factor donne la base,
	// qu'on redivise par le facteur de la cible.
	return Result{Value: value * fu.factor / tu.factor, FromSym: fu.sym, ToSym: tu.sym}, true, nil
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func pick(firstKnown bool, a, b string) string {
	if firstKnown {
		return a
	}
	return b
}

//  Températures 

// tempUnit ramène les écritures courantes à un code court (c/f/k) et dit si
// l'unité est bien une température.
func tempUnit(s string) (string, bool) {
	switch strings.TrimPrefix(s, "°") {
	case "c", "celsius":
		return "c", true
	case "f", "fahrenheit":
		return "f", true
	case "k", "kelvin":
		return "k", true
	}
	return "", false
}

func tempSym(u string) string {
	switch u {
	case "f":
		return "°F"
	case "k":
		return "K"
	default:
		return "°C"
	}
}

// On passe systématiquement par le Celsius comme pivot : deux conversions
// simples plutôt qu'un tableau de toutes les paires.
func toCelsius(v float64, u string) float64 {
	switch u {
	case "f":
		return (v - 32) * 5 / 9
	case "k":
		return v - 273.15
	default:
		return v
	}
}

func fromCelsius(c float64, u string) float64 {
	switch u {
	case "f":
		return c*9/5 + 32
	case "k":
		return c + 273.15
	default:
		return c
	}
}
