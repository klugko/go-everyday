package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Les classes de caractères d'un mot de passe.
const (
	lowers  = "abcdefghijklmnopqrstuvwxyz"
	uppers  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"
	symbols = "!@#$%^&*()-_=+[]{};:,.?"
)

// Caractères qui se confondent à l'œil ou selon la police : l/I/1, O/0…
const ambiguous = "lI1O0o`'\";:,."

// Policy décrit ce qu'on veut dans un mot de passe.
type Policy struct {
	Length      int
	Upper       bool
	Lower       bool
	Digits      bool
	Symbols     bool
	NoAmbiguous bool
}

// classes renvoie les jeux de caractères activés, déjà débarrassés des
// caractères ambigus si demandé. Chaque entrée est non vide.
func (p Policy) classes() []string {
	raw := []struct {
		on  bool
		set string
	}{
		{p.Lower, lowers},
		{p.Upper, uppers},
		{p.Digits, digits},
		{p.Symbols, symbols},
	}
	var out []string
	for _, c := range raw {
		if !c.on {
			continue
		}
		set := c.set
		if p.NoAmbiguous {
			set = removeAny(set, ambiguous)
		}
		if set != "" {
			out = append(out, set)
		}
	}
	return out
}

// pool est l'union de toutes les classes activées.
func (p Policy) pool() string {
	return strings.Join(p.classes(), "")
}

// GeneratePassword tire un mot de passe en garantissant au moins un
// caractère de chaque classe demandée, puis mélange le tout pour ne pas
// trahir la position des garanties.
func GeneratePassword(p Policy) (string, error) {
	classes := p.classes()
	if len(classes) == 0 {
		return "", errors.New("aucune classe de caractères activée")
	}
	if p.Length < len(classes) {
		return "", fmt.Errorf("longueur %d trop courte pour garantir %d classes", p.Length, len(classes))
	}
	pool := strings.Join(classes, "")

	out := make([]byte, p.Length)
	// Une garantie par classe…
	for i, set := range classes {
		out[i] = set[randInt(len(set))]
	}
	// …le reste pioché dans le pool complet.
	for i := len(classes); i < p.Length; i++ {
		out[i] = pool[randInt(len(pool))]
	}
	// Mélange de Fisher–Yates : sinon les premiers caractères suivraient
	// toujours l'ordre des classes.
	for i := len(out) - 1; i > 0; i-- {
		j := randInt(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return string(out), nil
}

// GeneratePassphrase assemble n mots tirés de la liste EFF.
func GeneratePassphrase(n int, sep string) (string, error) {
	if n <= 0 {
		return "", errors.New("il faut au moins un mot")
	}
	w := make([]string, n)
	for i := range w {
		w[i] = words[randInt(len(words))]
	}
	return strings.Join(w, sep), nil
}

// PasswordEntropy estime l'entropie en bits : log2(taille_du_pool ^
// longueur). C'est l'espace de recherche d'un attaquant qui connaît la
// politique mais pas le tirage — la mesure de référence.
func PasswordEntropy(p Policy) float64 {
	pool := len(p.pool())
	if pool == 0 {
		return 0
	}
	return float64(p.Length) * math.Log2(float64(pool))
}

// PassphraseEntropy : chaque mot apporte log2(taille_de_la_liste) bits.
func PassphraseEntropy(n int) float64 {
	if n <= 0 {
		return 0
	}
	return float64(n) * math.Log2(float64(len(words)))
}

// Strength traduit des bits d'entropie en verdict lisible. Les seuils
// suivent l'usage courant : sous ~28 bits ça tombe en quelques secondes,
// au-delà de 128 c'est hors d'atteinte d'une attaque par force brute.
func Strength(bits float64) string {
	switch {
	case bits < 28:
		return "très faible"
	case bits < 36:
		return "faible"
	case bits < 60:
		return "correct"
	case bits < 128:
		return "fort"
	default:
		return "excellent"
	}
}

// randInt renvoie un entier uniforme dans [0, n) via crypto/rand, sans
// biais de modulo (rand.Int rejette les tirages débordants pour nous).
// Un crypto/rand qui échoue, c'est un système cassé : il n'y a rien à
// rattraper, on panique.
func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic("crypto/rand indisponible : " + err.Error())
	}
	return int(v.Int64())
}

// removeAny renvoie s privé de tous les caractères présents dans cut.
func removeAny(s, cut string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(cut, r) {
			return -1
		}
		return r
	}, s)
}
