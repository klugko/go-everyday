package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	length := flag.Int("len", 20, "longueur du mot de passe")
	wordsN := flag.Int("words", 0, "mode passphrase : nombre de mots (>0 active le mode)")
	sep := flag.String("sep", "-", "séparateur entre les mots (passphrase)")
	count := flag.Int("n", 1, "nombre de propositions")

	noUpper := flag.Bool("no-upper", false, "exclure les majuscules")
	noLower := flag.Bool("no-lower", false, "exclure les minuscules")
	noDigits := flag.Bool("no-digits", false, "exclure les chiffres")
	noSymbols := flag.Bool("no-symbols", false, "exclure les symboles")
	noAmbiguous := flag.Bool("no-ambiguous", false, "exclure les caractères ambigus (l, I, 1, O, 0…)")
	quiet := flag.Bool("q", false, "ne pas afficher le score d'entropie")
	flag.Parse()

	var bits float64
	var gen func() (string, error)

	if *wordsN > 0 {
		bits = PassphraseEntropy(*wordsN)
		gen = func() (string, error) { return GeneratePassphrase(*wordsN, *sep) }
	} else {
		p := Policy{
			Length:      *length,
			Upper:       !*noUpper,
			Lower:       !*noLower,
			Digits:      !*noDigits,
			Symbols:     !*noSymbols,
			NoAmbiguous: *noAmbiguous,
		}
		bits = PasswordEntropy(p)
		gen = func() (string, error) { return GeneratePassword(p) }
	}

	for i := 0; i < *count; i++ {
		s, err := gen()
		if err != nil {
			fmt.Fprintln(os.Stderr, "pass:", err)
			os.Exit(1)
		}
		fmt.Println(s)
	}

	// Le score part sur stderr : le mot de passe reste seul sur stdout,
	// propre à rediriger ou copier-coller.
	if !*quiet {
		fmt.Fprintf(os.Stderr, "\n%.0f bits d'entropie — %s\n", bits, Strength(bits))
	}
}
