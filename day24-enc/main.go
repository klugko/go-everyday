// enc — chiffre ou déchiffre n'importe quel fichier (AES-256-GCM) pour le
// partager en sécurité. Par défaut on chiffre ; -d déchiffre.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	decrypt := flag.Bool("d", false, "déchiffrer (par défaut on chiffre)")
	out := flag.String("o", "", "fichier de sortie (déduit du nom sinon)")
	password := flag.String("p", "", "mot de passe (sinon $ENC_PASSWORD, sinon demandé)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "enc — chiffre/déchiffre un fichier avec un mot de passe.")
		fmt.Fprintln(os.Stderr, "usage : enc [-d] [-o sortie] [-p motdepasse] <fichier>")
		fmt.Fprintln(os.Stderr, "  chiffrer   : enc secret.pdf            → secret.pdf.enc")
		fmt.Fprintln(os.Stderr, "  déchiffrer : enc -d secret.pdf.enc     → secret.pdf")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *out, *password, *decrypt); err != nil {
		fmt.Fprintln(os.Stderr, "enc :", err)
		os.Exit(1)
	}
}

func run(in, out, password string, dec bool) error {
	pw, err := resolvePassword(password)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}

	var result []byte
	var dst string
	if dec {
		result, err = decrypt(data, pw)
		dst = outName(out, in, true)
	} else {
		result, err = encrypt(data, pw)
		dst = outName(out, in, false)
	}
	if err != nil {
		return err
	}

	// On refuse d'écraser un fichier existant : un mauvais sens (-d oublié) ne
	// doit pas détruire l'original.
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("%q existe déjà, je n'écrase pas", dst)
	}
	if err := os.WriteFile(dst, result, 0600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s → %s\n", in, dst)
	return nil
}

// outName décide du nom de sortie. En chiffrement on ajoute .enc ; en
// déchiffrement on le retire si présent, sinon on accole .dec pour ne jamais
// écraser la source par accident.
func outName(explicit, in string, dec bool) string {
	if explicit != "" {
		return explicit
	}
	if dec {
		if strings.HasSuffix(in, ".enc") {
			return strings.TrimSuffix(in, ".enc")
		}
		return in + ".dec"
	}
	return in + ".enc"
}

// resolvePassword cherche le mot de passe dans l'ordre : drapeau, environnement,
// saisie interactive. La variable d'environnement rend l'outil scriptable.
func resolvePassword(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if pw := os.Getenv("ENC_PASSWORD"); pw != "" {
		return pw, nil
	}
	fmt.Fprint(os.Stderr, "mot de passe : ")
	var pw string
	if _, err := fmt.Fscanln(os.Stdin, &pw); err != nil || pw == "" {
		return "", errors.New("mot de passe vide")
	}
	return pw, nil
}
