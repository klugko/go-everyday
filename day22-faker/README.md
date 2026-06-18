# day22 — faker

Remplir une base de démo à la main, c'est vite « Test Test », « aaa@bbb.cc » et
trois lignes copiées-collées. `faker` crache des fiches françaises plausibles —
noms, emails, téléphones, adresses — en CSV ou JSON, prêtes à seeder une base ou
à nourrir un test.

```
faker                                   # 10 fiches CSV (prenom,nom,email,tel,ville,cp)
faker -n 100 -o gens.csv                # 100 fiches dans un fichier
faker -f json -fields prenom,email      # JSON, colonnes choisies
faker -seed 42                          # rejoue exactement le même jeu
```

Champs disponibles : `prenom`, `nom`, `email`, `tel`, `adresse`, `ville`, `cp`,
`naissance`, `entreprise`. On en demande autant qu'on veut via `-fields`, dans
l'ordre où ils sortiront.

## Décisions qui ont compté

- **Une fiche est cohérente, pas un sac de champs.** On tire d'abord une
  `person` complète, puis on n'en projette que les colonnes demandées. C'est ce
  qui fait que l'email colle au nom (`lea.francois@…`) et que le code postal
  suit la ville au lieu de coller un CP marseillais sur une adresse parisienne.
  Ville et CP sont d'ailleurs stockés en paire dans le vivier, justement pour ne
  jamais les désynchroniser.
- **La graine rend tout reproductible.** Sans `-seed`, on part de l'horloge —
  chaque appel surprend. Avec une graine fixe, le même `faker -seed 42` redonne
  octet pour octet le même jeu : indispensable pour un test qui doit être
  déterministe. Tout passe par un seul `*rand.Rand` ; aucune source globale.
- **L'email se fabrique depuis le nom, accents retirés.** `Léa François`
  devient `lea.francois@…` : minuscules, accents aplatis, espaces en tirets. Un
  suffixe chiffré tombe une fois sur trois pour éviter que deux homonymes se
  retrouvent avec la même adresse dans une vraie base.
- **Les champs dérivés sont calculés, pas stockés.** Téléphone, adresse,
  date de naissance ne vivent pas dans la `person` : on les génère au moment où
  la colonne est demandée. Inutile de remplir une fiche entière quand on ne
  veut qu'un prénom.
- **Le JSON est écrit à la main.** Même raison que les autres jours qui sortent
  du JSON : une `map[string]any` se sérialiserait par clé triée et mélangerait
  l'ordre des colonnes. On parcourt donc `-fields` dans l'ordre et on n'emprunte
  `json.Marshal` que pour échapper proprement chaque clé et chaque valeur.
- **Un champ inconnu plante tout de suite.** `-fields prenom,xxx` renvoie une
  erreur avec la liste des champs valides, plutôt que de sortir une colonne
  `xxx` vide qu'on ne remarquerait qu'à la relecture.

## Ce que j'ai laissé tomber

- **Les viviers gigantesques.** Une trentaine de prénoms, autant de noms, une
  vingtaine de villes : largement assez de variété pour un jeu de test. Charger
  un dictionnaire complet n'apporterait rien et alourdirait le binaire.
- **La cohérence parfaite des dates.** La date de naissance est une date
  plausible (1950–2005, jour ≤ 28 pour ne jamais tomber sur un 30 février), pas
  un vrai calendrier. Un export de démo, pas un état civil.
- **Les locales autres que FR.** Le sujet, c'est de la donnée française. Ajouter
  l'anglais ou l'allemand, ce serait un deuxième vivier et un drapeau de plus —
  pas une refonte, mais hors périmètre du jour.
- **L'unicité garantie.** Rien n'empêche deux fiches identiques sur un gros
  tirage. Pour des tests, c'est même réaliste ; vouloir l'unicité, ce serait un
  autre outil.
