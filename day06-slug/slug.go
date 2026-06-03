package main

import (
	"strings"
	"unicode"
)

type Options struct {
	Sep   string 
	Lower bool   
	Max   int    
}

// Slugify transforme un titre en slug propre : translittération des
// accents, on ne garde que [a-z0-9], et tout le reste devient une
// frontière de mot.
//
//	"Café à Tana" -> "cafe-a-tana"
func Slugify(s string, opt Options) string {
	if opt.Sep == "" {
		opt.Sep = "-"
	}

	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	// emit avale une chaîne déjà translittérée : les caractères de slug
	// s'accumulent dans le mot courant, tout le reste le clôt.
	emit := func(chunk string) {
		for _, r := range chunk {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
				cur.WriteRune(r)
			default:
				flush()
			}
		}
	}

	for _, r := range s {
		// On cherche la translittération sur la rune minuscule pour ne
		// tenir qu'une seule entrée par lettre (é et É partagent "e").
		lr := unicode.ToLower(r)
		if repl, ok := translit[lr]; ok {
			if !opt.Lower && r != lr {
				repl = strings.ToUpper(repl)
			}
			emit(repl)
			continue
		}
		if opt.Lower {
			r = lr
		}
		emit(string(r))
	}
	flush()

	slug := strings.Join(words, opt.Sep)

	if opt.Max > 0 && len(slug) > opt.Max {
		slug = slug[:opt.Max]
		// On évite de couper en plein milieu d'un mot : on rogne jusqu'au
		// dernier séparateur s'il y en a un.
		if i := strings.LastIndex(slug, opt.Sep); i > 0 {
			slug = slug[:i]
		}
		slug = strings.Trim(slug, opt.Sep)
	}
	return slug
}

// translit fait correspondre les lettres accentuées et quelques
// caractères latins spéciaux à leur équivalent ASCII. Les clés sont en
// minuscules ; le casse de sortie est rétabli par l'appelant.
//
// Pourquoi une table à la main plutôt que golang.org/x/text/unicode/norm
// (NFD + suppression des diacritiques) ? La règle du dépôt, c'est zéro
// dépendance. Et pour du français — plus l'essentiel des langues
// européennes — une table explicite est lisible, sans surprise, et
// largement suffisante.
var translit = map[rune]string{
	'à': "a", 'â': "a", 'ä': "a", 'á': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ĩ': "i", 'ī': "i", 'ĭ': "i", 'į': "i", 'ı': "i",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ŏ': "o", 'ő': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ũ': "u", 'ū': "u", 'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'ç': "c", 'ć': "c", 'ĉ': "c", 'ċ': "c", 'č': "c",
	'ñ': "n", 'ń': "n", 'ņ': "n", 'ň': "n",
	'ý': "y", 'ÿ': "y", 'ŷ': "y",
	'ĝ': "g", 'ğ': "g", 'ġ': "g", 'ģ': "g",
	'ś': "s", 'ŝ': "s", 'ş': "s", 'š': "s",
	'ź': "z", 'ż': "z", 'ž': "z",
	'ł': "l", 'ļ': "l", 'ľ': "l",
	'ŕ': "r", 'ř': "r",
	'ţ': "t", 'ť': "t",
	'ď': "d", 'đ': "d", 'ð': "d",
	// Ligatures et lettres qui se rendent en deux caractères.
	'æ': "ae", 'œ': "oe", 'ß': "ss", 'þ': "th",
}
