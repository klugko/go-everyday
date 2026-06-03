# slug

Transformer un titre en slug d'URL, ça paraît être l'affaire de trois
lignes : minuscules, espaces en tirets, terminé. Puis on tape `Café à
Tana` et on obtient `caf-tana` parce que le `é` a sauté et le `à`,
caractère seul, a disparu dans la moulinette. Tout l'exercice tient dans
ce détail : **gérer les accents proprement**, en français d'abord.

`slug` prend un titre (en argument ou sur l'entrée standard) et rend un
identifiant propre : `cafe-a-tana`.

## Le cheminement

### Le vrai problème, c'est `é` → `e`

En théorie, retirer un accent c'est de la **normalisation Unicode** : on
décompose `é` en `e` + un accent combinant (forme NFD), puis on jette les
marques diacritiques. La bibliothèque `golang.org/x/text/unicode/norm`
fait exactement ça… mais c'est une dépendance externe, et la règle du
dépôt c'est zéro dépendance.

Du coup, table de translittération écrite à la main. C'est moins
« élégant » qu'un passage par NFD, mais pour du français — et la plupart
des langues européennes — une table explicite est lisible, prévisible, et
n'a aucune surprise au runtime. J'ai couvert toutes les voyelles
accentuées, les `ç`/`ñ`, et les cas qui se rendent en **deux lettres** :
`œ` → `oe`, `æ` → `ae`, `ß` → `ss`. Ce dernier point compte : un slug
n'est pas une suppression de caractères, c'est une *transcription*.

### Une seule passe, pas deux

Premier réflexe : translittérer toute la chaîne, puis re-parcourir pour
nettoyer. Mais les deux opérations sont la même boucle. Donc une passe :
pour chaque rune, soit on la translittère, soit on la garde ; et au fil de
l'eau, tout ce qui n'est pas `[a-z0-9]` clôt le mot courant. À la fin, on
joint les mots avec le séparateur. Conséquence agréable : les espaces
multiples, la ponctuation, les emojis — tout ce qui n'est pas reconnu
devient simplement une frontière. Pas de cas particulier à écrire pour
« deux espaces de suite » ou « tiret en trop ».

### La casse, ce petit piège

Pour ne pas dédoubler la table (`é` *et* `É`), je cherche toujours sur la
rune **minuscule**. Si l'utilisateur veut garder la casse (`-keep-case`),
je détecte que l'original était une majuscule et je remajuscule la
translittération — `École` donne bien `Ecole` et pas `ecole`.

### Couper sans mutiler

L'option `-max` limite la longueur, mais couper brutalement à N octets
laisse un mot tronqué moche (`the-quick-bro`). Je rogne donc jusqu'au
dernier séparateur : on perd un mot entier plutôt que d'en garder la
moitié. Et comme après nettoyage le slug est de l'ASCII pur, « octet » et
« caractère » coïncident — pas de souci de découpe en plein milieu d'un
caractère multioctet.

## Ce que j'ai laissé tomber

- **La normalisation Unicode complète (NFD).** Une dépendance pour gagner
  les langues que ma table ne couvre pas (cyrillique, grec, CJK…). Hors
  périmètre d'un slugifieur orienté français/européen, et un slug en
  alphabet latin reste de toute façon l'usage.
- **La translittération « intelligente » du CJK** (pinyin, romaji). C'est
  un projet à soi tout seul, pas une option d'un coin de table.
- **La déduplication / les suffixes `-2`.** Garantir l'unicité d'un slug,
  c'est le travail de la base de données qui le stocke, pas du
  transformateur de texte.

## Usage

```
slug [options] <titre>
cat titres.txt | slug [options]      # un slug par ligne
```

Options :

```
-sep <str>     séparateur entre les mots (déf. "-")
-max <n>       longueur maximale en octets (0 = sans limite)
-keep-case     préserver la casse au lieu de tout passer en minuscules
```

Exemples :

```
slug "Café à Tana"                 # cafe-a-tana
slug -sep _ "Crème brûlée"         # creme_brulee
slug -max 20 "Un titre à rallonge qui dépasse"
slug -keep-case "École Élémentaire"  # Ecole-Elementaire
```

## Organisation

```
main.go   CLI : flags, argument ou entrée standard
slug.go   Slugify + table de translittération des accents
```
