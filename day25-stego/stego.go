package main

import (
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
)

// Stéganographie LSB : on remplace le bit de poids faible de chaque canal R, G,
// B par un bit du message. Changer un canal de ±1 sur 255 est invisible à l'œil,
// et le PNG étant sans perte, les bits posés survivent au ré-encodage.
//
// Disposition : 4 octets de longueur (uint32, big-endian) puis le message.
// La longueur évite de deviner où s'arrêter à la lecture.

const headerBytes = 4

var (
	errTooBig = errors.New("message trop grand pour cette image")
	errNoMsg  = errors.New("aucun message caché lisible dans cette image")
)

// hide écrit msg dans une copie de l'image et la renvoie. On part toujours d'une
// NRGBA fraîche : un tableau d'octets R,G,B,A plat, facile à parcourir bit à
// bit, et on ne touche jamais à l'image d'origine.
func hide(src image.Image, msg []byte) (*image.NRGBA, error) {
	out := cloneNRGBA(src)

	payload := make([]byte, headerBytes+len(msg))
	binary.BigEndian.PutUint32(payload, uint32(len(msg)))
	copy(payload[headerBytes:], msg)

	// 3 canaux porteurs par pixel (on saute l'alpha, voir writeBits).
	capacity := out.Bounds().Dx() * out.Bounds().Dy() * 3
	if len(payload)*8 > capacity {
		return nil, errTooBig
	}

	writeBits(out.Pix, payload)
	return out, nil
}

// reveal relit le message caché. On lit d'abord l'en-tête de longueur, puis
// exactement ce nombre d'octets.
func reveal(src image.Image) ([]byte, error) {
	pix := toNRGBA(src).Pix

	head := readBits(pix, 0, headerBytes)
	n := int(binary.BigEndian.Uint32(head))

	// Une image sans message donne une longueur farfelue : on la borne par la
	// capacité réelle pour ne pas tenter de lire des mégaoctets de bruit.
	capacity := len(pix)/4*3 - headerBytes
	if n < 0 || n > capacity {
		return nil, errNoMsg
	}
	return readBits(pix, headerBytes, n), nil
}

// writeBits déverse les octets de data dans les LSB de pix, en sautant l'octet
// alpha (un canal sur quatre) pour ne pas trahir un fond transparent.
func writeBits(pix []byte, data []byte) {
	bit := 0
	total := len(data) * 8
	for i := 0; i < len(pix) && bit < total; i++ {
		if i%4 == 3 { // canal alpha : on n'y touche pas
			continue
		}
		b := (data[bit/8] >> (7 - bit%8)) & 1
		pix[i] = (pix[i] &^ 1) | b
		bit++
	}
}

// readBits fait l'inverse : il reconstruit n octets depuis les LSB, en
// commençant après skip octets déjà lus (l'en-tête).
func readBits(pix []byte, skip, n int) []byte {
	start := skip * 8
	out := make([]byte, n)
	bit := 0
	want := n * 8
	for i := 0; i < len(pix) && bit < start+want; i++ {
		if i%4 == 3 {
			continue
		}
		if bit >= start {
			pos := bit - start
			out[pos/8] |= (pix[i] & 1) << (7 - pos%8)
		}
		bit++
	}
	return out
}

// toNRGBA renvoie l'image sous forme NRGBA pour la lecture, sans copie inutile
// si elle l'est déjà. NRGBA est non prémultipliée : on lit R,G,B sans surprise.
func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	return cloneNRGBA(src)
}

// cloneNRGBA produit une copie NRGBA indépendante, quel que soit le type source.
func cloneNRGBA(src image.Image) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	draw.Draw(out, b, src, b.Min, draw.Src)
	return out
}
