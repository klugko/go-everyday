package main

import (
	_ "embed"
	"strings"
)

// La liste de mots EFF « large » (7776 mots, comme un jet de 5 dés), le
// standard de fait pour les passphrases diceware. On l'embarque dans le
// binaire : `embed` est de la stdlib, donc la règle « zéro dépendance »
// tient toujours.
//
// Source : EFF, "Deep Dive: New Wordlists for Random Passphrases".
//
//go:embed words.txt
var wordsRaw string

var words = strings.Fields(wordsRaw)
