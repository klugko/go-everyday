package main

// Correction d'erreur : Reed-Solomon sur le corps de Galois GF(256) 
func gfMul(x, y byte) byte {
	var z byte
	for i := 7; i >= 0; i-- {
		z = (z << 1) ^ ((z >> 7) * 0x1D)
		z ^= ((y >> uint(i)) & 1) * x
	}
	return z
}

//  construit le polynôme générateur de degré donné
func rsGenerator(degree int) []byte {
	result := make([]byte, degree)
	result[degree-1] = 1
	root := byte(1)
	for i := 0; i < degree; i++ {
		for j := 0; j < degree; j++ {
			result[j] = gfMul(result[j], root)
			if j+1 < degree {
				result[j] ^= result[j+1]
			}
		}
		root = gfMul(root, 0x02) // α suivant
	}
	return result
}

// renvoie les codewords de correction : le reste de la
// division du bloc de données par le générateur, dans GF(256).
func rsRemainder(data, gen []byte) []byte {
	result := make([]byte, len(gen))
	for _, b := range data {
		factor := b ^ result[0]
		copy(result, result[1:])
		result[len(result)-1] = 0
		for i := range result {
			result[i] ^= gfMul(gen[i], factor)
		}
	}
	return result
}

// La grille 
type Matrix struct {
	Size int
	mod  [][]bool
	fn   [][]bool
}

func newMatrix(size int) *Matrix {
	mod := make([][]bool, size)
	fn := make([][]bool, size)
	for i := range mod {
		mod[i] = make([]bool, size)
		fn[i] = make([]bool, size)
	}
	return &Matrix{Size: size, mod: mod, fn: fn}
}

// pour l'affichage.
func (m *Matrix) Dark(r, c int) bool { return m.mod[r][c] }


func (m *Matrix) setFn(r, c int, dark bool) {
	if r < 0 || r >= m.Size || c < 0 || c >= m.Size {
		return
	}
	m.mod[r][c] = dark
	m.fn[r][c] = true
}

//  pose tout ce qui n'est pas données 
func (m *Matrix) drawFunctionPatterns(ver int, level Level) {
	size := m.Size
	m.drawFinder(3, 3)
	m.drawFinder(3, size-4)
	m.drawFinder(size-4, 3)

	// Lignes de timing : alternance entre les motifs de détection.
	for i := 8; i < size-8; i++ {
		dark := i%2 == 0
		m.setFn(6, i, dark)
		m.setFn(i, 6, dark)
	}

	pos := alignPositions[ver]
	for _, r := range pos {
		for _, c := range pos {
			// Écarter ceux qui chevaucheraient un motif de détection.
			if (r == 6 && c == 6) || (r == 6 && c == size-7) || (r == size-7 && c == 6) {
				continue
			}
			m.drawAlign(r, c)
		}
	}

	if ver >= 7 {
		m.drawVersion(ver)
	}
	// Réserve les modules de format (et pose le module toujours noir).
	m.drawFormat(level, 0)
}

// détection 7×7
func (m *Matrix) drawFinder(cr, cc int) {
	for dr := -4; dr <= 4; dr++ {
		for dc := -4; dc <= 4; dc++ {
			d := max(abs(dr), abs(dc))
			m.setFn(cr+dr, cc+dc, d != 2 && d != 4)
		}
	}
}

// pose un motif d'alignement 5×5 centré sur (cr,cc) 
func (m *Matrix) drawAlign(cr, cc int) {
	for dr := -2; dr <= 2; dr++ {
		for dc := -2; dc <= 2; dc++ {
			m.setFn(cr+dr, cc+dc, max(abs(dr), abs(dc)) != 1)
		}
	}
}


func (m *Matrix) drawVersion(ver int) {
	rem := ver
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	bits := ver<<12 | rem
	size := m.Size
	for i := 0; i < 18; i++ {
		b := getBit(bits, i)
		a := size - 11 + i%3
		c := i / 3
		m.setFn(a, c, b) // en bas à gauche
		m.setFn(c, a, b) // en haut à droite
	}
}

// pose les deux copies de l'info de format 
func (m *Matrix) drawFormat(level Level, mask int) {
	data := formatBits[level]<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	bits := (data<<10 | rem) ^ 0x5412

	// Copie 1, autour de l'œil en haut à gauche.
	for i := 0; i <= 5; i++ {
		m.setFn(i, 8, getBit(bits, i))
	}
	m.setFn(7, 8, getBit(bits, 6))
	m.setFn(8, 8, getBit(bits, 7))
	m.setFn(8, 7, getBit(bits, 8))
	for i := 9; i < 15; i++ {
		m.setFn(8, 14-i, getBit(bits, i))
	}
	// Copie 2, répartie sous les deux autres yeux. 
	size := m.Size
	for i := 0; i < 8; i++ {
		m.setFn(8, size-1-i, getBit(bits, i))
	}
	for i := 0; i < 7; i++ {
		m.setFn(size-1-i, 8, getBit(bits, 14-i))
	}
	m.setFn(size-8, 8, true) // module toujours noir
}

//  déroule le flux de bits dans la grille selon le parcours
func (m *Matrix) drawCodewords(data []byte) {
	size := m.Size
	i := 0
	for right := size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5 // sauter la colonne de timing
		}
		for vert := 0; vert < size; vert++ {
			for j := 0; j < 2; j++ {
				col := right - j
				upward := (right+1)&2 == 0
				row := vert
				if upward {
					row = size - 1 - vert
				}
				if !m.fn[row][col] && i < len(data)*8 {
					m.mod[row][col] = getBit(int(data[i>>3]), 7-(i&7))
					i++
				}
			}
		}
	}
}

// inverse les modules de données selon une des 8 formules de
// masque.
func (m *Matrix) applyMask(mask int) {
	for r := 0; r < m.Size; r++ {
		for c := 0; c < m.Size; c++ {
			if m.fn[r][c] {
				continue
			}
			var inv bool
			switch mask {
			case 0:
				inv = (r+c)%2 == 0
			case 1:
				inv = r%2 == 0
			case 2:
				inv = c%3 == 0
			case 3:
				inv = (r+c)%3 == 0
			case 4:
				inv = (r/2+c/3)%2 == 0
			case 5:
				inv = r*c%2+r*c%3 == 0
			case 6:
				inv = (r*c%2+r*c%3)%2 == 0
			case 7:
				inv = ((r+c)%2+r*c%3)%2 == 0
			}
			if inv {
				m.mod[r][c] = !m.mod[r][c]
			}
		}
	}
}

// essaie les 8 masques, garde celui dont le motif est jugé
// le plus lisible (pénalité minimale), et l'applique pour de bon avec
// l'info de format correspondante.
func (m *Matrix) applyBestMask(level Level) int {
	best, bestScore := 0, 1<<31
	for mask := 0; mask < 8; mask++ {
		m.applyMask(mask)
		m.drawFormat(level, mask)
		if s := m.penalty(); s < bestScore {
			best, bestScore = mask, s
		}
		m.applyMask(mask) // annule
	}
	m.applyMask(best)
	m.drawFormat(level, best)
	return best
}

// note la grille selon les quatre règles du standard.
func (m *Matrix) penalty() int {
	size := m.Size
	score := 0

	// Règle 1 : séries de 5 modules ou plus de même couleur.
	for r := 0; r < size; r++ {
		score += lineRuns(m.mod[r])
	}
	for c := 0; c < size; c++ {
		score += lineRuns(m.column(c))
	}

	// Règle 2 : blocs 2×2 d'une seule couleur.
	for r := 0; r < size-1; r++ {
		for c := 0; c < size-1; c++ {
			v := m.mod[r][c]
			if m.mod[r][c+1] == v && m.mod[r+1][c] == v && m.mod[r+1][c+1] == v {
				score += 3
			}
		}
	}

	// Règle 3 : motif ressemblant à un œil de détection (1:1:3:1:1).
	for r := 0; r < size; r++ {
		score += finderLike(m.mod[r])
	}
	for c := 0; c < size; c++ {
		score += finderLike(m.column(c))
	}

	// Règle 4 : écart de la proportion de noir par rapport à 50 %, par
	// tranches de 5 %. Le calcul en entiers reproduit exactement le
	// floor(|pourcentage - 50| / 5) du standard, sans flottant.
	dark := 0
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if m.mod[r][c] {
				dark++
			}
		}
	}
	total := size * size
	score += 10 * (abs(dark*100-50*total) / (5 * total))
	return score
}

func (m *Matrix) column(c int) []bool {
	out := make([]bool, m.Size)
	for r := 0; r < m.Size; r++ {
		out[r] = m.mod[r][c]
	}
	return out
}

func lineRuns(line []bool) int {
	s, run := 0, 1
	for i := 1; i < len(line); i++ {
		if line[i] == line[i-1] {
			run++
			continue
		}
		if run >= 5 {
			s += 3 + (run - 5)
		}
		run = 1
	}
	if run >= 5 {
		s += 3 + (run - 5)
	}
	return s
}

// Le cœur du motif pénalisé : la suite 1:1:3:1:1 (sombre, clair, triple
// sombre, clair, sombre) sur 7 modules.
var finderCore = [7]bool{true, false, true, true, true, false, true}

// finderLike compte les occurrences du cœur 1:1:3:1:1 bordé, d'un côté au
// moins, par quatre modules clairs (ou par le bord du symbole, où la zone
// de silence joue ce rôle). Chacune vaut 40 points. La recherche reprend
// après le motif quand il compte, ou quatre modules plus loin sinon, pour
// attraper les cœurs qui se chevauchent.
func finderLike(line []bool) int {
	n := len(line)
	s := 0
	for i := findCore(line, 0); i != -1; {
		after := i + 7
		if i == 0 || i == n-7 || !anyDark(line, i-4, i) || !anyDark(line, after, after+4) {
			s += 40
			i = findCore(line, after)
		} else {
			i = findCore(line, i+4)
		}
	}
	return s
}

// findCore renvoie l'indice du prochain cœur 1:1:3:1:1 à partir de start,
// ou -1.
func findCore(line []bool, start int) int {
	for i := start; i+7 <= len(line); i++ {
		match := true
		for k := 0; k < 7; k++ {
			if line[i+k] != finderCore[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// indique s'il existe un module sombre dans [lo, hi), bornes
// ramenées dans la ligne.
func anyDark(line []bool, lo, hi int) bool {
	if lo < 0 {
		lo = 0
	}
	if hi > len(line) {
		hi = len(line)
	}
	for i := lo; i < hi; i++ {
		if line[i] {
			return true
		}
	}
	return false
}
