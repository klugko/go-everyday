// stego — cache un message texte dans une image PNG, ou l'en ressort, sans
// qu'on voie de différence à l'œil. Méthode LSB, voir stego.go.
package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "stego :", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "stego — cache (ou révèle) un message dans un PNG.")
	fmt.Fprintln(os.Stderr, "usage :")
	fmt.Fprintln(os.Stderr, "  stego hide <source.png> <sortie.png> [message]   (message lu sur stdin si absent)")
	fmt.Fprintln(os.Stderr, "  stego reveal <image.png>")
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("commande manquante")
	}
	switch args[0] {
	case "hide":
		return cmdHide(args[1:])
	case "reveal":
		return cmdReveal(args[1:])
	default:
		usage()
		return fmt.Errorf("commande inconnue %q", args[0])
	}
}

func cmdHide(args []string) error {
	if len(args) < 2 {
		return errors.New("usage : stego hide <source.png> <sortie.png> [message]")
	}
	srcPath, outPath := args[0], args[1]

	var msg []byte
	if len(args) >= 3 {
		msg = []byte(strings.Join(args[2:], " "))
	} else {
		// Pas de message en argument : on le lit sur stdin (utile pour un fichier
		// entier : stego hide in.png out.png < lettre.txt).
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		msg = raw
	}
	if len(msg) == 0 {
		return errors.New("message vide")
	}

	src, err := readPNG(srcPath)
	if err != nil {
		return err
	}
	out, err := hide(src, msg)
	if err != nil {
		return err
	}
	if err := writePNG(outPath, out); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d octet(s) cachés dans %s\n", len(msg), outPath)
	return nil
}

func cmdReveal(args []string) error {
	if len(args) != 1 {
		return errors.New("usage : stego reveal <image.png>")
	}
	src, err := readPNG(args[0])
	if err != nil {
		return err
	}
	msg, err := reveal(src)
	if err != nil {
		return err
	}
	// Le message brut sur stdout : redirigeable vers un fichier sans ajout.
	os.Stdout.Write(msg)
	return nil
}

func readPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s : PNG illisible (%w)", path, err)
	}
	return img, nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}
