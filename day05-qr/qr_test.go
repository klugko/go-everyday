package main

import (
	"reflect"
	"testing"
)

// --- GF(256) ---

func TestGFMul(t *testing.T) {
	cases := []struct {
		x, y, want byte
	}{
		{0, 5, 0},      // 0 absorbe
		{1, 200, 200},  // 1 neutre
		{2, 2, 4},      // pas encore de réduction
		{2, 128, 0x1D}, // 256 réduit modulo 0x11D donne 0x1D
	}
	for _, c := range cases {
		if got := gfMul(c.x, c.y); got != c.want {
			t.Errorf("gfMul(%d,%d) = %d, attendu %d", c.x, c.y, got, c.want)
		}
		// La multiplication est commutative.
		if gfMul(c.x, c.y) != gfMul(c.y, c.x) {
			t.Errorf("gfMul non commutatif sur (%d,%d)", c.x, c.y)
		}
	}
}

// --- Encodage des données (avant correction) ---

func TestBuildDataCodewords(t *testing.T) {
	// "A" (0x41) en version 1, niveau M : un cas qu'on peut dérouler à la
	// main. Mode 0100, compteur 00000001, donnée 01000001, terminateur
	// 0000 → 0x40 0x14 0x10, puis bourrage 0xEC/0x11 jusqu'à 16 octets.
	want := []byte{
		0x40, 0x14, 0x10,
		0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11, 0xEC,
	}
	got := buildDataCodewords([]byte("A"), 1, M)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildDataCodewords =\n %v\nattendu\n %v", got, want)
	}
}

func TestAddEccSingleBlock(t *testing.T) {
	// Version 1 = un seul bloc : l'entrelacement se réduit à
	// données suivies de correction. On vérifie la longueur totale et que
	// les données restent en tête, intactes.
	data := buildDataCodewords([]byte("A"), 1, M)
	full := addEccAndInterleave(data, 1, M)
	if len(full) != totalCodewords[1] {
		t.Fatalf("longueur = %d, attendu %d", len(full), totalCodewords[1])
	}
	if !reflect.DeepEqual(full[:len(data)], data) {
		t.Fatalf("les données ne sont pas en tête : %v", full[:len(data)])
	}
}

// --- Choix de version ---

func TestEncodeChoosesVersion(t *testing.T) {
	// 1 octet tient en version 1 (taille 21).
	m, err := Encode([]byte("A"), M)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 21 {
		t.Fatalf("taille = %d, attendu 21", m.Size)
	}
	// 20 octets ne tiennent plus en version 1/M, il faut la version 2 (25).
	m, err = Encode([]byte("01234567890123456789"), M)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != 25 {
		t.Fatalf("taille = %d, attendu 25", m.Size)
	}
}

func TestEncodeTooLong(t *testing.T) {
	big := make([]byte, 1000) // au-delà de la capacité de la version 10
	if _, err := Encode(big, H); err == nil {
		t.Fatal("attendu une erreur de dépassement")
	}
}

// --- Structure de la grille ---

func TestFunctionPatterns(t *testing.T) {
	m, err := Encode([]byte("A"), M)
	if err != nil {
		t.Fatal(err)
	}
	// Œil de détection en haut à gauche : bordure noire, anneau blanc.
	if !m.Dark(0, 0) || !m.Dark(0, 6) || !m.Dark(6, 0) {
		t.Error("bordure de l'œil de détection manquante")
	}
	if m.Dark(1, 1) {
		t.Error("l'anneau de l'œil devrait être blanc en (1,1)")
	}
	if !m.Dark(2, 2) {
		t.Error("le cœur de l'œil devrait être noir en (2,2)")
	}
	// Séparateur blanc juste à côté de l'œil.
	if m.Dark(7, 0) {
		t.Error("le séparateur en (7,0) devrait être blanc")
	}
	// Ligne de timing : alternance, noir aux coordonnées paires.
	if !m.Dark(6, 8) || m.Dark(6, 9) {
		t.Error("ligne de timing incorrecte")
	}
	// Module toujours noir, en (taille-8, 8).
	if !m.Dark(m.Size-8, 8) {
		t.Error("le module toujours noir est absent")
	}
}

// --- Wi-Fi ---

func TestWifiPayload(t *testing.T) {
	got := WifiPayload("MonReseau", "secret", "WPA", false)
	want := "WIFI:T:WPA;S:MonReseau;P:secret;;"
	if got != want {
		t.Fatalf("got %q, attendu %q", got, want)
	}
}

func TestWifiPayloadEscapesAndNopass(t *testing.T) {
	// Caractères spéciaux échappés, et pas de champ P sur un réseau ouvert.
	got := WifiPayload("Café;Bar", "", "nopass", true)
	want := `WIFI:T:nopass;S:Café\;Bar;H:true;;`
	if got != want {
		t.Fatalf("got %q, attendu %q", got, want)
	}
}

// Golden : la grille complète d'« HELLO » en niveau M. La sortie a été
// vérifiée décodable par un lecteur de référence (zxing) ; ce test la fige
// pour attraper toute régression dans l'encodage, la pose ou le masquage.
func TestEncodeGolden(t *testing.T) {
	want := []string{
		"#######.#..#..#######",
		"#.....#.####..#.....#",
		"#.###.#...#.#.#.###.#",
		"#.###.#.#.#.#.#.###.#",
		"#.###.#....#..#.###.#",
		"#.....#....##.#.....#",
		"#######.#.#.#.#######",
		"........#..##........",
		"#.##.###.#.##.#..#.##",
		".##.##.#.######..##..",
		"#...#.#..#.#.......##",
		"#.##...#...#..####.#.",
		".#.######...#..#..#.#",
		"........####..#...#.#",
		"#######.#..##..#.....",
		"#.....#.#.#....#####.",
		"#.###.#.....######.##",
		"#.###.#.#.##..#.####.",
		"#.###.#.##..#.##..#..",
		"#.....#...#..#.##...#",
		"#######.#.#..#.#.....",
	}
	m, err := Encode([]byte("HELLO"), M)
	if err != nil {
		t.Fatal(err)
	}
	if m.Size != len(want) {
		t.Fatalf("taille %d, attendu %d", m.Size, len(want))
	}
	for r := 0; r < m.Size; r++ {
		var b []byte
		for c := 0; c < m.Size; c++ {
			if m.Dark(r, c) {
				b = append(b, '#')
			} else {
				b = append(b, '.')
			}
		}
		if string(b) != want[r] {
			t.Errorf("ligne %d:\n got %s\n want %s", r, b, want[r])
		}
	}
}

func TestParseEnc(t *testing.T) {
	for in, want := range map[string]string{
		"wpa": "WPA", "WPA2": "WPA", "wep": "WEP", "nopass": "nopass", "open": "nopass",
	} {
		got, err := parseEnc(in)
		if err != nil || got != want {
			t.Errorf("parseEnc(%q) = %q, %v ; attendu %q", in, got, err, want)
		}
	}
	if _, err := parseEnc("rot13"); err == nil {
		t.Error("attendu une erreur pour un chiffrement inconnu")
	}
}
