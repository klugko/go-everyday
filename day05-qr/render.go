package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"
)

// La zone de silence (marge claire obligatoire) vaut 4 modules dans le
// standard. Au terminal on se contente de 2 : chaque module y prend deux
// caractères de large, alors 4 de marge de chaque côté gonflent vite la
// largeur au point de dépasser l'écran et de casser la lecture. Dans le
// PNG, où la largeur n'est pas contrainte, on garde les 4 réglementaires.
const (
	quietTerminal = 2
	quietPNG      = 4
)

// RenderTerminal dessine le QR avec des couleurs ANSI : fond clair pour
// les modules blancs, fond sombre pour les noirs, deux espaces par module.
// On force les couleurs plutôt que de réutiliser le thème du terminal —
// sinon un fond sombre rendrait la marge invisible et le code illisible
// pour un scanner. Deux espaces donnent un module à peu près carré, la
// cellule texte étant environ deux fois plus haute que large.
func RenderTerminal(m *Matrix, w io.Writer) {
	const (
		light = "\x1b[107m" // fond blanc vif
		dark  = "\x1b[40m"  // fond noir
		reset = "\x1b[0m"
	)
	for r := -quietTerminal; r < m.Size+quietTerminal; r++ {
		var b strings.Builder
		cur := ""
		for c := -quietTerminal; c < m.Size+quietTerminal; c++ {
			color := light
			if r >= 0 && r < m.Size && c >= 0 && c < m.Size && m.Dark(r, c) {
				color = dark
			}
			if color != cur {
				b.WriteString(color)
				cur = color
			}
			b.WriteString("  ")
		}
		b.WriteString(reset)
		fmt.Fprintln(w, b.String())
	}
}

// WritePNG écrit le QR en image, chaque module agrandi à scale pixels,
// entouré de la zone de silence. Image en niveaux de gris : un QR n'a
// besoin que de noir et de blanc.
func WritePNG(m *Matrix, path string, scale int) error {
	if scale < 1 {
		scale = 1
	}
	dim := (m.Size + quietPNG*2) * scale
	img := image.NewGray(image.Rect(0, 0, dim, dim))
	for i := range img.Pix {
		img.Pix[i] = 0xff // tout blanc, on ne peint que le noir ensuite
	}
	black := color.Gray{Y: 0}
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if !m.Dark(r, c) {
				continue
			}
			x0, y0 := (c+quietPNG)*scale, (r+quietPNG)*scale
			for y := y0; y < y0+scale; y++ {
				for x := x0; x < x0+scale; x++ {
					img.SetGray(x, y, black)
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// WifiPayload assemble la chaîne « WIFI: » que les téléphones savent
// interpréter pour se connecter sans saisir le mot de passe. enc vaut
// WPA, WEP ou nopass (réseau ouvert).
func WifiPayload(ssid, pass, enc string, hidden bool) string {
	var b strings.Builder
	b.WriteString("WIFI:")
	b.WriteString("T:" + enc + ";")
	b.WriteString("S:" + wifiEscape(ssid) + ";")
	if enc != "nopass" {
		b.WriteString("P:" + wifiEscape(pass) + ";")
	}
	if hidden {
		b.WriteString("H:true;")
	}
	b.WriteString(";")
	return b.String()
}

// wifiEscape protège les caractères spéciaux du format : un SSID ou un mot
// de passe contenant ; , : " ou \ casserait sinon la chaîne.
func wifiEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', ';', ',', ':', '"':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
