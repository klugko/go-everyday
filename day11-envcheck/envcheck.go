package main

import "strings"

// Diff range les clés d'un côté et de l'autre : ce qui manque dans le
// .env par rapport au modèle, et ce qui s'y trouve en plus.
type Diff struct {
	Missing []string 
	Extra   []string 
}

// OK quand le .env colle exactement au modèle.
func (d Diff) OK() bool { return len(d.Missing) == 0 && len(d.Extra) == 0 }

// Keys lit un contenu de fichier .env et rend les clés dans l'ordre où
// elles apparaissent, sans doublon. On se moque des valeurs : comparer
// des clés suffit à dire ce qui manque ou ce qui traîne.
func Keys(content string) []string {
	var keys []string
	seen := map[string]bool{}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Lignes vides et commentaires : rien à voir ici.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Forme "export, on retire le préfixe.
		line = strings.TrimPrefix(line, "export ")

		// Sans "=", on ignore.
		name, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)

		// Une clé avec un espace est suspecte (phrase, faute de frappe) :
		// on ne la compte pas pour ne pas polluer le diff.
		if name == "" || strings.ContainsAny(name, " \t") {
			continue
		}
		if !seen[name] {
			seen[name] = true
			keys = append(keys, name)
		}
	}
	return keys
}

// Compare confronte les clés du modèle à celles du .env. Chaque liste
// garde l'ordre de son fichier d'origine : on lit le diff comme on
// lirait les fichiers.
func Compare(exampleKeys, envKeys []string) Diff {
	inEnv := setOf(envKeys)
	inExample := setOf(exampleKeys)

	var d Diff
	for _, k := range exampleKeys {
		if !inEnv[k] {
			d.Missing = append(d.Missing, k)
		}
	}
	for _, k := range envKeys {
		if !inExample[k] {
			d.Extra = append(d.Extra, k)
		}
	}
	return d
}

func setOf(keys []string) map[string]bool {
	s := make(map[string]bool, len(keys))
	for _, k := range keys {
		s[k] = true
	}
	return s
}
