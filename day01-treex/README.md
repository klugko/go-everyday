# treex

Le problème c'est que `tree` dans un projet Node, ça crache 50000 lignes de
`node_modules/`. À chaque fois faut faire `tree -I 'node_modules|dist|...'`
et on en oublie toujours. Or git, lui, sait déjà ce qu'il faut ignorer
c'est dans `.gitignore`. Donc l'idée bête : un `tree` qui lit le `.gitignore`.

## Le cheminement

Premier réflexe : `filepath.WalkDir`, c'est ce qu'on a dans la stdlib pour
parcourir des dossiers. Sauf qu'on a besoin de deux trucs en même temps :

- additionner les tailles depuis les feuilles vers le parent
- couper net un sous-arbre si le dossier est ignoré (sans même lire dedans)

Avec `WalkDir` faut une map `parent → size` qu'on remplit après coup, et
appeler `SkipDir` au bon moment. Ça marche mais c'est moche. Une fonction
récursive qui renvoie `(*node, size)`, ça fait la même chose en plus court
et plus clair. Donc récursion maison.

Ensuite, matcher du `.gitignore`. Premier réflexe encore : `filepath.Match`,
c'est dans la stdlib. Sauf qu'il connaît pas `**` (genre `**/*.tmp`), et il
distingue pas un pattern ancré (`/build`) d'un flottant (`build`). Donc il
manque la moitié de la sémantique git. Du coup deuxième idée : on compile
chaque ligne du `.gitignore` en regexp. Une fois compilées c'est rapide,
et la conversion tient en 30 lignes.

Le truc qui m'a fait réfléchir un peu plus longtemps, c'est le nesting.
Un `.gitignore` à la racine peut ignorer `*.txt`, et un `.gitignore` plus
bas dans l'arbre peut faire `!important.txt` pour le ré-inclure. Première
intuition : un objet `frame` par `.gitignore`, et une pile de frames qu'on
empile/dépile en descendant. Ça marche mais c'est lourd.

Puis je me suis dit : si je mets toutes les règles à plat dans une seule
slice parent d'abord, enfant après — et que je dis "dernier match gagne",
ça donne exactement la bonne sémantique git, sans aucune notion de stack.
La slice EST la pile. Et comme en Go on passe `rules []rule` à la fonction
récursive et qu'on fait `rules = append(rules, ...)` à chaque niveau, le
parent ne voit jamais les règles ajoutées par ses enfants. Free.

Le reste c'est de l'évidence : tri dossiers-puis-fichiers pour la lisibilité,
toujours sauter `.git/` (personne veut le voir), formatage `KiB/MiB`, et
caractères box-drawing pour le rendu.

## Ce que j'ai laissé tomber

Au début je supportais les classes `[abc]` dans les patterns, les
échappements `\#` `\!`, le fichier global `~/.config/git/ignore`. Sauf
que dans les vrais `.gitignore` que je rencontre, ces trucs apparaissent
genre jamais. Donc ciseaux.

Pas de suivi des symlinks non plus  risque de boucle infinie, et c'est
pas ce qu'on veut quand on visualise un arbre.

## Usage

```
go run . [-L profondeur] [-a] [--no-gitignore] [chemin]
```

## Organisation

```
main.go          flags, entrée CLI
gitignore.go     parser + matcher
tree.go          descente récursive + rendu
```

Plus les tests. Le test qui m'a le plus servi c'est `TestNestedGitignore` :
c'est lui qui prouve que la pile-à-plat marche. Tant qu'il passe, je sais
que la sémantique est bonne.
