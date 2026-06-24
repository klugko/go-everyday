package main

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// fixture fabrique une image colorée déterministe assez grande pour porter un
// message d'essai.
func fixture(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), uint8(x + y), 255})
		}
	}
	return img
}

func TestHideRevealRoundTrip(t *testing.T) {
	msg := []byte("rendez-vous à minuit, porte B — surtout pas un mot.")
	out, err := hide(fixture(64, 64), msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reveal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("message altéré : %q != %q", got, msg)
	}
}

// L'image porteuse ne doit changer qu'imperceptiblement : chaque canal d'au plus
// ±1 par rapport à l'original.
func TestImageBarelyChanges(t *testing.T) {
	src := fixture(64, 64)
	out, err := hide(src, []byte("petit secret"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range src.Pix {
		d := int(src.Pix[i]) - int(out.Pix[i])
		if d < -1 || d > 1 {
			t.Fatalf("canal %d a bougé de %d (max ±1 attendu)", i, d)
		}
	}
}

// hide ne doit pas modifier l'image d'origine.
func TestSourceUntouched(t *testing.T) {
	src := fixture(32, 32)
	before := append([]byte(nil), src.Pix...)
	if _, err := hide(src, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, src.Pix) {
		t.Fatal("hide a modifié l'image source")
	}
}

func TestMessageTooBig(t *testing.T) {
	// 4x4 = 16 pixels × 3 canaux = 48 bits = 6 octets, en-tête compris.
	if _, err := hide(fixture(4, 4), bytes.Repeat([]byte("A"), 100)); err != errTooBig {
		t.Fatalf("attendu errTooBig, obtenu %v", err)
	}
}

// Une image sans rien de caché ne doit pas inventer un message géant.
func TestNoMessage(t *testing.T) {
	// Image toute noire : l'en-tête de longueur lu vaut 0 → message vide, pas
	// d'erreur ; on vérifie surtout qu'on ne plante pas et qu'on ne lit pas
	// des mégaoctets de bruit.
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	got, err := reveal(img)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("attendu message vide, obtenu %d octets", len(got))
	}
}

func TestBinaryMessage(t *testing.T) {
	msg := []byte{0x00, 0xff, 0x10, 0x80, 0x7f, 0x00}
	out, err := hide(fixture(32, 32), msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reveal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("octets binaires altérés : %v != %v", got, msg)
	}
}
