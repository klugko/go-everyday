// read — analyse un texte : mots, temps de lecture, score de lisibilité.
package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
	"unicode"
)

func main() {
	wpm := flag.Int("wpm", 200, "vitesse de lecture, mots par minute")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "read — analyse la lisibilité d'un texte.")
		fmt.Fprintln(os.Stderr, "usage : read [--wpm n] [fichier]")
		fmt.Fprintln(os.Stderr, "sans fichier, lit l'entrée standard.")
		flag.PrintDefaults()
	}
	flag.Parse()

	text, err := readInput(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "read :", err)
		os.Exit(1)
	}

	s := analyze(text)
	if s.Words == 0 {
		fmt.Fprintln(os.Stderr, "read : texte vide")
		os.Exit(1)
	}
	fmt.Print(report(s, *wpm))
}

// readInput : le fichier passé en argument, ou stdin si on n'a rien — pratique en pipe.
func readInput(path string) (string, error) {
	if path == "" {
		data, err := io.ReadAll(os.Stdin)
		return string(data), err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// stats regroupe les comptages bruts dont tout le reste se déduit.
type stats struct {
	Words     int
	Sentences int
	Syllables int
}

// analyze range tout dans stats : les mots et syllabes par champ, les phrases
// par balayage du texte. Compter les phrases sur le mot ratait le cas français
// "aujourd'hui !" où le point d'exclamation est détaché par une espace.
func analyze(text string) stats {
	var s stats
	for _, word := range strings.Fields(text) {
		if !hasLetter(word) {
			continue
		}
		s.Words++
		s.Syllables += syllables(word)
	}
	s.Sentences = countSentences(text)
	if s.Words > 0 && s.Sentences == 0 {
		s.Sentences = 1
	}
	return s
}

// countSentences compte les groupes de ponctuation finale. "Vraiment ?!" ou
// "fini..." valent une phrase chacun : une suite de terminateurs ne compte qu'une fois.
func countSentences(text string) int {
	n, prevTerm := 0, false
	for _, r := range text {
		term := r == '.' || r == '!' || r == '?' || r == '…'
		if term && !prevTerm {
			n++
		}
		prevTerm = term
	}
	return n
}

// syllables approxime les syllabes en comptant les groupes de voyelles.
// "bateau" -> ba-teau = 2, "rythme" -> 1. Grossier mais suffisant pour un score.
func syllables(word string) int {
	n, prevVowel := 0, false
	for _, r := range strings.ToLower(word) {
		v := isVowel(r)
		if v && !prevVowel {
			n++
		}
		prevVowel = v
	}
	if n == 0 {
		return 1 
	}
	return n
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'y',
		'à', 'â', 'ä', 'é', 'è', 'ê', 'ë', 'î', 'ï', 'ô', 'ö', 'ù', 'û', 'ü':
		return true
	}
	return false
}

func hasLetter(word string) bool {
	for _, r := range word {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// flesch : score de lisibilité de Flesch (Reading Ease). Plus c'est haut, plus
// c'est facile. La formule pénalise les phrases longues et les mots à rallonge.
func flesch(s stats) float64 {
	w, sent, syl := float64(s.Words), float64(s.Sentences), float64(s.Syllables)
	return 206.835 - 1.015*(w/sent) - 84.6*(syl/w)
}

// label traduit le score en mot du quotidien, bornes classiques de Flesch.
func label(score float64) string {
	switch {
	case score >= 90:
		return "très facile"
	case score >= 70:
		return "facile"
	case score >= 50:
		return "moyen"
	case score >= 30:
		return "difficile"
	default:
		return "très difficile"
	}
}

// readingTime : durée de lecture arrondie à la minute supérieure, jamais zéro.
func readingTime(words, wpm int) time.Duration {
	minutes := math.Ceil(float64(words) / float64(wpm))
	return time.Duration(minutes) * time.Minute
}

// report met en forme le bilan. On clampe le score à [0, 100] pour l'affichage :
// la formule peut déborder sur des textes extrêmes, mais la note ne le fait pas.
func report(s stats, wpm int) string {
	score := math.Max(0, math.Min(100, flesch(s)))
	var b strings.Builder
	fmt.Fprintf(&b, "mots            %d\n", s.Words)
	fmt.Fprintf(&b, "phrases         %d\n", s.Sentences)
	fmt.Fprintf(&b, "mots/phrase     %.1f\n", float64(s.Words)/float64(s.Sentences))
	fmt.Fprintf(&b, "temps de lecture %s\n", fmtDuration(readingTime(s.Words, wpm)))
	fmt.Fprintf(&b, "lisibilité      %.0f/100 (%s)\n", score, label(score))
	return b.String()
}

// fmtDuration : "3 min" ou "1 h 05 min", sans les secondes qui n'apportent rien.
func fmtDuration(d time.Duration) string {
	m := int(d.Minutes())
	if m < 60 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %02d min", m/60, m%60)
}
