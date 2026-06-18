package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// person regroupe une fiche cohérente. On la fabrique d'un bloc puis on n'en
// projette que les colonnes demandées : comme ça l'email colle au nom, et le
// code postal à la ville, au lieu de tirer chaque champ dans son coin.
type person struct {
	prenom, nom string
	ville, cp   string
}

// champs reconnus, dans l'ordre où on les sortira si l'utilisateur en demande
// plusieurs. Sert aussi à valider -fields et à afficher l'aide.
var champs = []string{
	"prenom", "nom", "email", "tel", "adresse", "ville", "cp",
	"naissance", "entreprise",
}

// newPerson tire une fiche complète à partir d'un générateur donné. Tout part
// d'ici : on fixe le couple ville/CP une fois pour toutes.
func newPerson(r *rand.Rand) person {
	v := villes[r.Intn(len(villes))]
	return person{
		prenom: prenoms[r.Intn(len(prenoms))],
		nom:    noms[r.Intn(len(noms))],
		ville:  v.nom,
		cp:     v.cp,
	}
}

// field renvoie la valeur d'une colonne pour une fiche. Les champs dérivés
// (email, tél, adresse…) sont calculés ici plutôt que stockés : inutile de
// remplir une fiche entière quand on ne demande qu'un prénom.
func field(p person, name string, r *rand.Rand) string {
	switch name {
	case "prenom":
		return p.prenom
	case "nom":
		return p.nom
	case "email":
		return email(p, r)
	case "tel":
		return phone(r)
	case "adresse":
		return fmt.Sprintf("%d %s", r.Intn(200)+1, rues[r.Intn(len(rues))])
	case "ville":
		return p.ville
	case "cp":
		return p.cp
	case "naissance":
		// Date plausible d'un adulte : entre 1950 et 2005.
		return fmt.Sprintf("%02d/%02d/%d", r.Intn(28)+1, r.Intn(12)+1, r.Intn(56)+1950)
	case "entreprise":
		return entreprises[r.Intn(len(entreprises))]
	}
	return ""
}

// email construit une adresse à partir du nom : prenom.nom@domaine, en
// minuscules et sans accent. Un suffixe chiffré de temps en temps évite que
// deux "Jean Martin" se marchent dessus dans une vraie base.
func email(p person, r *rand.Rand) string {
	local := strings.ToLower(deaccent(p.prenom) + "." + deaccent(p.nom))
	if r.Intn(3) == 0 {
		local += fmt.Sprintf("%d", r.Intn(90)+10)
	}
	return local + "@" + domaines[r.Intn(len(domaines))]
}

// phone génère un mobile français : 06 ou 07 suivi de huit chiffres, groupés
// par deux comme on les écrit ici.
func phone(r *rand.Rand) string {
	d := make([]byte, 10)
	d[0] = '0'
	d[1] = byte('6' + r.Intn(2))
	for i := 2; i < 10; i++ {
		d[i] = byte('0' + r.Intn(10))
	}
	s := string(d)
	return s[0:2] + " " + s[2:4] + " " + s[4:6] + " " + s[6:8] + " " + s[8:10]
}

// deaccent retire les accents courants pour fabriquer un email propre.
// Volontairement minimal : juste ce qu'on croise dans des prénoms/noms FR.
var accents = strings.NewReplacer(
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"à", "a", "â", "a", "ä", "a",
	"î", "i", "ï", "i", "ô", "o", "ö", "o",
	"û", "u", "ü", "u", "ù", "u", "ç", "c",
	" ", "-", "'", "",
)

func deaccent(s string) string {
	return accents.Replace(strings.ToLower(s))
}
