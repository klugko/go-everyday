package main

import "strings"

// lang décrit comment reconnaître les commentaires d'un langage. C'est tout ce
// dont on a besoin pour séparer code, commentaire et lignes vides.
type lang struct {
	name       string
	line       []string // débuts de commentaire sur une ligne (// , #, --)
	blockStart string   // début de bloc (/* …), vide si le langage n'en a pas
	blockEnd   string   // fin de bloc (… */)
}

// On range les langages par extension. Volontairement court : les plus courants
// suffisent, et en ajouter un, c'est une ligne. Plusieurs extensions peuvent
// pointer vers le même langage.
var byExt = map[string]lang{}

func register(l lang, exts ...string) {
	for _, e := range exts {
		byExt[e] = l
	}
}

func init() {
	cStyle := func(name string) lang {
		return lang{name: name, line: []string{"//"}, blockStart: "/*", blockEnd: "*/"}
	}
	hashStyle := func(name string) lang {
		return lang{name: name, line: []string{"#"}}
	}

	register(cStyle("Go"), ".go")
	register(cStyle("C"), ".c", ".h")
	register(cStyle("C++"), ".cpp", ".cc", ".hpp")
	register(cStyle("Java"), ".java")
	register(cStyle("Rust"), ".rs")
	register(cStyle("JavaScript"), ".js", ".mjs")
	register(cStyle("TypeScript"), ".ts", ".tsx")
	register(lang{name: "CSS", blockStart: "/*", blockEnd: "*/"}, ".css")

	register(hashStyle("Python"), ".py")
	register(hashStyle("Ruby"), ".rb")
	register(hashStyle("Shell"), ".sh", ".bash")
	register(hashStyle("YAML"), ".yml", ".yaml")
	register(hashStyle("TOML"), ".toml")

	register(lang{name: "SQL", line: []string{"--"}, blockStart: "/*", blockEnd: "*/"}, ".sql")
	register(lang{name: "Lua", line: []string{"--"}, blockStart: "--[[", blockEnd: "]]"}, ".lua")

	// HTML/Markdown : pas de notion simple de « commentaire de ligne ». On les
	// compte quand même, tout passe en code sauf les blocs <!-- … -->.
	register(lang{name: "HTML", blockStart: "<!--", blockEnd: "-->"}, ".html", ".htm")
	register(lang{name: "Markdown"}, ".md")
}

// count sépare un contenu en (vides, commentaires, code). Règle assumée, comme
// le vrai cloc : une ligne qui mêle code et commentaire compte pour du code.
func count(content string, l lang) (blank, comment, code int) {
	// On enlève le seul saut de ligne final pour ne pas inventer une ligne vide
	// fantôme ; les vraies lignes vides internes, elles, restent comptées.
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return 0, 0, 0
	}

	inBlock := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))

		if inBlock {
			comment++
			if l.blockEnd != "" && strings.Contains(line, l.blockEnd) {
				inBlock = false
			}
			continue
		}
		switch {
		case line == "":
			blank++
		case hasAnyPrefix(line, l.line):
			comment++
		case l.blockStart != "" && strings.HasPrefix(line, l.blockStart):
			comment++
			// Bloc ouvert mais pas refermé sur la même ligne : on reste dedans.
			if !strings.Contains(line[len(l.blockStart):], l.blockEnd) {
				inBlock = true
			}
		default:
			code++
		}
	}
	return blank, comment, code
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
