package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	digits := flag.Int("digits", 6, "nombre de chiffres du code")
	period := flag.Int("period", 30, "durée de validité en secondes")
	algoName := flag.String("algo", "SHA1", "algorithme : SHA1, SHA256 ou SHA512")
	watch := flag.Bool("watch", false, "rafraîchir le code en continu")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage : totp <secret base32 | otpauth://...> [options]")
		os.Exit(2)
	}
	arg := flag.Arg(0)

	// Deux entrées possibles : une URI otpauth complète (qui porte ses
	// propres paramètres), ou juste un secret base32 piloté par les flags.
	var cfg Config
	var err error
	if strings.HasPrefix(arg, "otpauth://") {
		cfg, err = ParseURI(arg)
	} else {
		var secret []byte
		secret, err = DecodeSecret(arg)
		cfg = Config{Secret: secret, Digits: *digits, Period: *period}
		if err == nil {
			cfg.Algo, err = ParseAlgo(*algoName)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "totp:", err)
		os.Exit(1)
	}

	if *watch {
		watchLoop(cfg)
		return
	}

	now := time.Now().Unix()
	code := TOTP(cfg.Secret, now, cfg.Period, cfg.Digits, cfg.Algo)
	rem := cfg.Period - int(now%int64(cfg.Period))
	fmt.Printf("%s  (encore %ds)\n", code, rem)
}

// watchLoop réimprime le code et une barre de progression chaque seconde,
// sur la même ligne. Ctrl-C pour sortir.
func watchLoop(cfg Config) {
	for {
		now := time.Now().Unix()
		code := TOTP(cfg.Secret, now, cfg.Period, cfg.Digits, cfg.Algo)
		rem := cfg.Period - int(now%int64(cfg.Period))
		fmt.Printf("\r%s  %s %2ds ", code, bar(rem, cfg.Period), rem)
		time.Sleep(time.Second)
	}
}

// bar dessine une barre qui se vide à mesure que le code expire.
func bar(rem, period int) string {
	const width = 20
	full := rem * width / period
	return "[" + strings.Repeat("#", full) + strings.Repeat("-", width-full) + "]"
}
