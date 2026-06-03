# pass

Un générateur de mots de passe, c'est facile à écrire et facile à écrire
*mal*. Le piège n'est pas l'algorithme, c'est le détail qui le rend sûr ou
illusoire : la source d'aléa. `pass` génère deux choses — des mots de
passe aléatoires et des passphrases façon diceware — et affiche pour
chacune un **score d'entropie** honnête.

## Le cheminement

### `crypto/rand`, jamais `math/rand`

C'est la seule décision qui compte vraiment. `math/rand` est prévisible :
qui connaît la graine rejoue toute la suite. Pour un mot de passe, c'est
disqualifiant. Donc `crypto/rand` partout.

Mais tirer un caractère dans un pool, c'est choisir un index dans
`[0, n)`, et le réflexe `rand_byte % n` introduit un **biais de modulo** :
si 256 n'est pas multiple de `n`, les premiers indices sortent un peu plus
souvent. Sur un alphabet de 95 caractères, le biais est réel. La parade,
c'est le rejet des tirages qui débordent — et `crypto/rand.Int` le fait
déjà pour nous. Une ligne, zéro biais. Un `crypto/rand` qui échoue
signifie un système cassé : il n'y a rien à rattraper, je panique plutôt
que de retourner un mot de passe silencieusement faible.

### Garantir les classes sans tricher sur l'aléa

« Au moins une majuscule, un chiffre, un symbole » — les règles classiques.
L'implémentation naïve (tirer, vérifier, recommencer si une classe manque)
fonctionne mais gaspille des tirages. Je place plutôt **un caractère
garanti par classe demandée**, je complète depuis le pool entier, puis je
**mélange** le tout (Fisher–Yates, toujours sur `crypto/rand`). Sans ce
mélange, les premiers caractères suivraient toujours l'ordre des classes —
une régularité qu'un attaquant exploiterait. Et si la longueur demandée
est plus petite que le nombre de classes, je refuse : on ne peut pas
garantir quatre classes sur trois caractères.

### Les passphrases, et pourquoi une liste de 7776 mots

Un mot de passe fort est imprononçable ; une passphrase est mémorisable
*et* forte — à condition de tirer les mots au hasard, pas de les choisir.
J'embarque la **liste EFF « large »** : 7776 mots (exactement 6⁵, un mot
par combinaison de cinq dés, d'où le nom « diceware »), choisis pour être
courants, distincts à l'écrit et sans préfixes ambigus. Chaque mot apporte
log₂(7776) ≈ **12,9 bits**. Six mots, c'est ~77 bits : hors de portée
d'une attaque par force brute, et ça se retient.

Le fichier est embarqué dans le binaire avec `//go:embed`. C'est de la
stdlib, donc la règle « zéro dépendance » tient toujours — pas de fichier
à trimballer à côté de l'exécutable.

### L'entropie, dite honnêtement

Le score affiché, c'est `log₂(taille_du_pool ^ longueur)` pour un mot de
passe, et `mots × log₂(7776)` pour une passphrase. C'est l'espace de
recherche d'un attaquant qui **connaît la politique** (longueur, classes,
liste) mais ignore le tirage — la mesure de référence, ni gonflée ni
faussement rassurante. Le verdict (« faible », « fort »…) suit les seuils
usuels : sous ~28 bits ça tombe en quelques secondes, au-delà de 128 c'est
illusoire d'essayer.

Détail d'ergonomie : le mot de passe sort sur **stdout**, le score sur
**stderr**. Comme ça `pass > secret.txt` n'écrit que le mot de passe, prêt
à copier-coller, sans le commentaire.

## Ce que j'ai laissé tomber

- **Un analyseur type zxcvbn** (détection de motifs « P@ssw0rd »,
  séquences clavier, dates). Pertinent pour *juger* un mot de passe choisi
  par un humain ; inutile ici, où l'aléa est garanti dès la génération.
- **Les règles de composition exotiques** (« exactement 2 chiffres », « pas
  deux symboles d'affilée »). Elles réduisent l'entropie en croyant
  l'augmenter. Les toggles par classe suffisent.
- **La copie automatique dans le presse-papier.** Dépendant de l'OS, et un
  simple `| clip` (Windows) ou `| pbcopy` (macOS) fait le travail.

## Usage

```
pass [options]                 # mot de passe aléatoire
pass -words <n> [options]      # passphrase de n mots
```

Options :

```
-len <n>         longueur du mot de passe (déf. 20)
-words <n>       mode passphrase : nombre de mots
-sep <str>       séparateur entre les mots (déf. "-")
-n <n>           générer n propositions
-no-upper        exclure les majuscules
-no-lower        exclure les minuscules
-no-digits       exclure les chiffres
-no-symbols      exclure les symboles
-no-ambiguous    exclure les caractères ambigus (l, I, 1, O, 0…)
-q               ne pas afficher le score d'entropie
```

Exemples :

```
pass                              # 20 caractères, toutes classes
pass -len 32 -no-symbols          # 32 caractères alphanumériques
pass -words 6                     # passphrase de 6 mots
pass -n 5 -no-ambiguous           # 5 candidats, sans caractères piégeux
pass -words 4 -sep . -q           # 4 mots séparés par des points, sans score
```

## Organisation

```
main.go    CLI : flags, aiguillage mot de passe / passphrase
pass.go    génération (crypto/rand), entropie, score de robustesse
words.go   embarque la liste de mots
words.txt  liste EFF « large » (7776 mots)
```
