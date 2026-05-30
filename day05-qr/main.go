package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	ec := flag.String("ec", "M", "niveau de correction : L, M, Q ou H")
	pngPath := flag.String("png", "", "écrit aussi un PNG à ce chemin")
	scale := flag.Int("scale", 8, "taille d'un module en pixels (PNG)")

	wifi := flag.Bool("wifi", false, "mode Wi-Fi : encode des identifiants de connexion")
	ssid := flag.String("ssid", "", "Wi-Fi : nom du réseau (SSID)")
	pass := flag.String("pass", "", "Wi-Fi : mot de passe")
	enc := flag.String("enc", "WPA", "Wi-Fi : WPA, WEP ou nopass")
	hidden := flag.Bool("hidden", false, "Wi-Fi : réseau masqué (SSID non diffusé)")
	flag.Parse()

	level, err := parseLevel(*ec)
	if err != nil {
		fatal(err)
	}

	payload, err := buildPayload(*wifi, *ssid, *pass, *enc, *hidden, flag.Args())
	if err != nil {
		fatal(err)
	}

	m, err := Encode([]byte(payload), level)
	if err != nil {
		fatal(err)
	}

	RenderTerminal(m, os.Stdout)
	if *pngPath != "" {
		if err := WritePNG(m, *pngPath, *scale); err != nil {
			fatal(err)
		}
		fmt.Fprintf(os.Stderr, "PNG écrit : %s (%d×%d modules)\n", *pngPath, m.Size, m.Size)
	}
}

// décide quoi encoder : soit des identifiants Wi-Fi, soit le
// texte passé en argument.
func buildPayload(wifi bool, ssid, pass, enc string, hidden bool, args []string) (string, error) {
	if wifi {
		if ssid == "" {
			return "", fmt.Errorf("-wifi : -ssid est requis")
		}
		e, err := parseEnc(enc)
		if err != nil {
			return "", err
		}
		if e != "nopass" && pass == "" {
			return "", fmt.Errorf("-wifi : -pass requis (ou -enc nopass pour un réseau ouvert)")
		}
		return WifiPayload(ssid, pass, e, hidden), nil
	}
	if len(args) == 0 {
		return "", fmt.Errorf("rien à encoder : passe un texte en argument, ou utilise -wifi")
	}
	return strings.Join(args, " "), nil
}

func parseLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "L":
		return L, nil
	case "M":
		return M, nil
	case "Q":
		return Q, nil
	case "H":
		return H, nil
	}
	return 0, fmt.Errorf("niveau de correction inconnu : %q (attendu L, M, Q ou H)", s)
}

func parseEnc(s string) (string, error) {
	switch strings.ToLower(s) {
	case "wpa", "wpa2", "wpa3":
		return "WPA", nil
	case "wep":
		return "WEP", nil
	case "nopass", "none", "open", "":
		return "nopass", nil
	}
	return "", fmt.Errorf("chiffrement Wi-Fi inconnu : %q (attendu WPA, WEP ou nopass)", s)
}

func fatal(args ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"qr:"}, args...)...)
	os.Exit(1)
}
