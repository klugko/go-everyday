# renamer

Renommer un fichier, c'est facile. En renommer trois cents — les photos
d'un voyage, une série d'exports numérotés n'importe comment, des fichiers
pleins d'espaces — c'est là qu'on ouvre un tableur ou qu'on bricole une
boucle PowerShell qu'on regrette. `renamer` fait ça par règles : tu
décris la transformation une fois, tu **vois le résultat avant**, et tu
appliques seulement si ça te va.

## Le cheminement

Le point de départ, c'est la peur. Renommer en masse, c'est l'opération
où une regex un peu trop gourmande transforme 300 fichiers en bouillie
d'un coup, sans Ctrl-Z. Donc la première décision n'est pas technique :
**par défaut, l'outil ne touche à rien.** Il calcule, il affiche `old →
new`, et il s'arrête. Il faut `-apply` pour qu'il agisse. L'aperçu n'est
pas une option de confort, c'est le mode normal.

### Trois règles qui se composent

L'objectif demandait regex, numérotation et casse. Plutôt que trois
outils, une seule chaîne appliquée dans un ordre fixe :

1. **find / replace** — une regex sur le nom. `-replace` accepte `$1`,
   `$2` pour les groupes capturés (la syntaxe native de `regexp`).
2. **case** — `lower`, `upper` ou `title`.
3. **num** — un gabarit avec `{n}` (le compteur) et `{name}` (le nom issu
   des étapes précédentes), ex `"{n}-{name}"`.

L'ordre n'est pas arbitraire : on nettoie d'abord (regex), on normalise la
casse ensuite, et on numérote ce qui en ressort. Numéroter avant de
nettoyer n'aurait pas de sens — le `{n}` se ferait manger par la regex.

### Le radical, pas l'extension

Toutes les transformations ne touchent que le **radical** du nom, jamais
l'extension. `PHOTO.JPG` passé en minuscules donne `photo.JPG`, pas
`photo.jpg`. C'est ce qu'on veut neuf fois sur dix : personne ne range ses
fichiers en cassant leur `.jpg`. Et la numérotation s'insère dans le nom,
jamais après l'extension. Si un jour je veux vraiment changer une
extension, je le ferai à la main — ce n'est pas le métier d'un renommeur
en masse.

### Le pad automatique

`-num "{n}-{name}"` sur douze fichiers donne `01-`, `02-`, …, `12-`. La
largeur du numéro se cale toute seule sur le plus grand : c'est ce qui
fait que les noms se trient correctement dans l'explorateur (`02` avant
`10`). On peut forcer avec `-pad`, décaler le départ avec `-start`,
changer le pas avec `-step`.

### Les conflits, attrapés avant d'agir

C'est le cœur du truc. Avant de renommer quoi que ce soit, on construit le
plan complet et on cherche ce qui clocherait :

- un nom qui deviendrait **vide** ;
- **deux sources qui visent la même cible** — l'une écraserait l'autre ;
- une **cible qui existe déjà** sur le disque sans faire partie du lot.

Si un seul conflit traîne, **rien** n'est renommé, même avec `-apply`. On
liste les problèmes et on sort. Mieux vaut un refus net qu'une perte
silencieuse.

### Le renommage en deux temps

Reste un piège classique : l'**échange**. Renommer `a → b` et `b → a` dans
la foulée, naïvement, écrase `b` à la première étape. Pareil pour un cycle
`a → b → c → a`. La parade tient en deux passes : on déplace d'abord tout
vers des noms temporaires, puis des temporaires vers les noms finaux. Plus
aucune collision possible en chemin, et les échanges marchent. Et si la
première passe échoue, on remet en place ce qui avait bougé.

Bonus Windows : un renommage qui ne change que la casse (`photo` →
`Photo`) ferait croire à une "cible déjà existante" sur un système de
fichiers insensible à la casse. On compare via `os.SameFile` — si la
cible *est* la source, ce n'est pas un conflit.

## Ce que j'ai laissé tomber

- **La récursion.** Un dossier passé en argument, ce sont ses enfants
  directs, pas tout l'arbre. Renommer en masse se raisonne dossier par
  dossier ; descendre récursivement multiplie les surprises.
- **Le tri configurable pour la numérotation.** L'ordre est alphabétique
  sur le chemin, point. Suffisant pour le cas courant (un dossier, des
  noms réguliers). Trier par date aurait demandé un flag de plus pour un
  gain rare.
- **Annuler (`undo`).** L'aperçu obligatoire et la détection de conflits
  couvrent l'essentiel du risque. Un vrai undo voudrait dire journaliser
  chaque opération — disproportionné ici.
- **Les symlinks.** Ignorés, comme les autres jours : renommer une cible
  via un alias est piégeux.

## Usage

```
renamer [règles] [chemins...]      # aperçu (ne touche à rien)
renamer [règles] -apply [chemins]  # applique
```

Sans chemin, c'est le dossier courant. Les chemins peuvent être des
fichiers ou des dossiers (enfants directs). `-glob` filtre par nom — utile
sous PowerShell, qui ne développe pas `*.jpg` pour un exe.

Règles :

```
-find <regex>     motif cherché dans le nom (hors extension)
-replace <str>    remplacement, $1 $2 pour les groupes
-case <mode>      lower | upper | title
-num <gabarit>    ex "{n}-{name}"  ({n} = compteur, {name} = nom courant)
-start <n>        premier numéro (déf. 1)
-step <n>         pas (déf. 1)
-pad <n>          largeur du numéro, 0 = auto (déf. 0)
-glob <pattern>   filtre, ex *.jpg
-apply            applique réellement
```

Exemples :

```
# Nettoyer des espaces et passer en minuscules, voir l'aperçu
renamer -find "\s+" -replace "_" -case lower -glob "*.JPG" ./photos

# Renuméroter une série proprement : 01-, 02-, ...
renamer -num "{n}-{name}" ./scans -apply

# Repartir d'un nom neuf et tout renuméroter
renamer -num "vacances-{n}" -start 1 -pad 3 ./album -apply
```

Aperçu type :

```
  DSC_0421.JPG  →  01-dsc_0421.JPG
  DSC 0422.JPG  →  02-dsc_0422.JPG

2 à renommer, 0 inchangés.

Aperçu seulement. Relance avec -apply pour renommer.
```

## Organisation

```
main.go        CLI : flags, validation, enchaînement
renamer.go     gather, transform, plan + conflits, renommage en deux temps
```
