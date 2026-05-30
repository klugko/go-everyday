package main

import "fmt"

// Génération d'un QR code à partir de zéro, en mode octet (byte mode),
// sans aucune dépendance — c'est la règle du repo, et c'est aussi le seul
// moyen de vraiment comprendre ce qu'est un QR code. Tout se joue en
// quatre temps : encoder les données en bits, leur coller de la
// correction d'erreur Reed-Solomon, poser le tout dans une grille selon
// un parcours en zigzag, puis choisir le masque qui rend le motif le plus
// lisible. La référence claire sur le sujet est l'implémentation de
// Nayuki ; les algorithmes ci-dessous en reprennent la structure.

// Level : niveau de correction d'erreur. Plus on monte, plus le code
// résiste aux taches et aux plis, mais moins il reste de place pour les
// données (à version égale).
type Level int

const (
	L Level = iota // ~7 % récupérable
	M              // ~15 % — le défaut, bon compromis
	Q              // ~25 %
	H              // ~30 %
)

// On se limite aux versions 1 à 10. C'est un choix assumé : la version 10
// encaisse déjà ~210 octets en niveau M — largement de quoi loger une URL
// ou un mot de passe Wi-Fi, le cœur de cet outil. Monter jusqu'à 40
// voudrait dire embarquer cinq fois plus de tables pour des cas qu'on ne
// rencontre jamais en pratique au terminal.
const maxVersion = 10

// Tables du standard ISO/IEC 18004, indexées [niveau][version]. Pour
// chaque couple on n'a besoin que de deux nombres : combien de codewords
// de correction par bloc, et combien de blocs. L'algorithme
// d'entrelacement en déduit seul la taille de chaque bloc, y compris les
// cas où les blocs n'ont pas tous la même longueur.
var eccPerBlock = [4][maxVersion + 1]int{
	L: {0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18},
	M: {0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26},
	Q: {0, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24},
	H: {0, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28},
}

var numBlocks = [4][maxVersion + 1]int{
	L: {0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4},
	M: {0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5},
	Q: {0, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8},
	H: {0, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8},
}

// Nombre total de codewords (données + correction) par version. Cette
// valeur découle de la géométrie de la grille, mais la lister est plus
// lisible que de la recalculer.
var totalCodewords = [maxVersion + 1]int{0, 26, 44, 70, 100, 134, 172, 196, 242, 292, 346}

// Centres des motifs d'alignement par version. Les combinaisons (ligne,
// colonne) donnent leurs positions ; celles qui tomberaient sur un motif
// de détection sont écartées au moment de la pose.
var alignPositions = [maxVersion + 1][]int{
	{}, {}, {6, 18}, {6, 22}, {6, 26}, {6, 30},
	{6, 34}, {6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50},
}

// Code du niveau de correction tel qu'il apparaît dans l'info de format
// (2 bits). L'ordre n'est pas L<M<Q<H : c'est celui du standard.
var formatBits = [4]int{L: 1, M: 0, Q: 3, H: 2}

func levelName(l Level) string { return []string{"L", "M", "Q", "H"}[l] }

// numDataCodewords : place réellement disponible pour les données, une
// fois la correction d'erreur retranchée.
func numDataCodewords(ver int, level Level) int {
	return totalCodewords[ver] - numBlocks[level][ver]*eccPerBlock[level][ver]
}

// Encode transforme des octets en grille de modules prête à afficher.
func Encode(data []byte, level Level) (*Matrix, error) {
	ver, err := chooseVersion(len(data), level)
	if err != nil {
		return nil, err
	}
	cw := addEccAndInterleave(buildDataCodewords(data, ver, level), ver, level)
	m := newMatrix(ver*4 + 17)
	m.drawFunctionPatterns(ver, level)
	m.drawCodewords(cw)
	m.applyBestMask(level)
	return m, nil
}

// chooseVersion prend la plus petite version où les données tiennent. Le
// nombre de bits du compteur de caractères dépend de la version (8 bits
// jusqu'à la 9, 16 ensuite), d'où le calcul refait à chaque essai.
func chooseVersion(n int, level Level) (int, error) {
	for v := 1; v <= maxVersion; v++ {
		need := 4 + countBits(v) + 8*n
		if need <= numDataCodewords(v, level)*8 {
			return v, nil
		}
	}
	maxBytes := numDataCodewords(maxVersion, level) - 3 // ~ octets en version 10
	return 0, fmt.Errorf("données trop longues : %d octets, max ~%d en version %d niveau %s",
		n, maxBytes, maxVersion, levelName(level))
}

// countBits : largeur du compteur de caractères en mode octet.
func countBits(ver int) int {
	if ver >= 10 {
		return 16
	}
	return 8
}

// buildDataCodewords sérialise les données en codewords : indicateur de
// mode, compteur, octets bruts, puis terminateur et bourrage jusqu'à
// remplir la capacité de la version.
func buildDataCodewords(data []byte, ver int, level Level) []byte {
	capacity := numDataCodewords(ver, level) * 8
	var w bitWriter
	w.write(0b0100, 4) // mode octet
	w.write(len(data), countBits(ver))
	for _, b := range data {
		w.write(int(b), 8)
	}
	// Terminateur : jusqu'à 4 zéros, sans déborder la capacité.
	if t := capacity - w.n(); t > 0 {
		if t > 4 {
			t = 4
		}
		w.write(0, t)
	}
	// Aligner sur l'octet.
	for w.n()%8 != 0 {
		w.write(0, 1)
	}
	// Bourrage : 0xEC et 0x11 en alternance, le motif imposé par le
	// standard pour combler ce qui reste.
	pad := []int{0xEC, 0x11}
	for i := 0; w.n() < capacity; i = (i + 1) % 2 {
		w.write(pad[i], 8)
	}
	return w.bytes()
}

// addEccAndInterleave découpe les codewords en blocs, calcule la
// correction Reed-Solomon de chacun, puis entrelace données et correction
// colonne par colonne. L'entrelacement est ce qui rend le code robuste à
// une tache localisée : un dégât concentré se répartit alors sur
// plusieurs blocs, dont chacun peut le corriger.
func addEccAndInterleave(data []byte, ver int, level Level) []byte {
	nBlocks := numBlocks[level][ver]
	eccLen := eccPerBlock[level][ver]
	raw := totalCodewords[ver]
	shortLen := raw / nBlocks         // longueur des blocs courts
	numShort := nBlocks - raw%nBlocks // combien de blocs courts

	gen := rsGenerator(eccLen)
	// Chaque bloc fait shortLen+1 cases : les blocs courts laissent un
	// trou (case 0) entre données et correction, qu'on saute à
	// l'entrelacement. Ça évite de gérer deux longueurs partout.
	blocks := make([][]byte, nBlocks)
	k := 0
	for i := 0; i < nBlocks; i++ {
		datLen := shortLen - eccLen
		if i >= numShort {
			datLen++ // blocs longs : un codeword de données de plus
		}
		dat := data[k : k+datLen]
		k += datLen
		blk := make([]byte, shortLen+1)
		copy(blk, dat)
		copy(blk[shortLen+1-eccLen:], rsRemainder(dat, gen))
		blocks[i] = blk
	}

	result := make([]byte, 0, raw)
	for i := 0; i < shortLen+1; i++ {
		for j := 0; j < nBlocks; j++ {
			// Sauter le trou des blocs courts.
			if i != shortLen-eccLen || j >= numShort {
				result = append(result, blocks[j][i])
			}
		}
	}
	return result
}

// bitWriter accumule des bits dans l'ordre où on les écrit, puis les
// reconditionne en octets. Un []bool est plus lisible qu'une gymnastique
// de masques, et la quantité de bits reste petite.
type bitWriter struct{ bits []bool }

func (w *bitWriter) write(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, (v>>uint(i))&1 == 1)
	}
}

func (w *bitWriter) n() int { return len(w.bits) }

func (w *bitWriter) bytes() []byte {
	out := make([]byte, len(w.bits)/8)
	for i, b := range w.bits {
		if b {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func getBit(x, i int) bool { return (x>>uint(i))&1 != 0 }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
