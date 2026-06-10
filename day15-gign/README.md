# day15 — gign

Un `.gitignore` pour ta stack, tout de suite. Tu démarres un projet Go avec un
front Node et VS Code par-dessus : `gign go node vscode` te crache le fichier
prêt à coller, sans aller copier-coller trois pages d'un site.

## Le problème

Le premier fichier d'un dépôt, c'est presque toujours le `.gitignore`. Et à
chaque fois on retourne sur gitignore.io ou on fouille un vieux projet pour
récupérer les mêmes lignes. Or ce sont *toujours* les mêmes : les binaires Go,
le `node_modules/`, le `__pycache__/`… Autant les avoir sous la main, dans un
binaire, hors-ligne.

```
gign go                        # le .gitignore Go, sur la sortie standard
gign go node vscode            # plusieurs stacks d'un coup
gign -o .gitignore go node     # écrit directement dans le fichier
gign --list                    # qu'est-ce que je peux demander ?
```

Plusieurs stacks se cumulent, chacune sous son titre :

```
### Go ###
*.exe
...

### Node ###
node_modules/
...
```

## Décisions qui ont compté

- **Les templates voyagent dans le binaire.** `//go:embed templates/*.gitignore`
  les colle dans l'exécutable : on copie `gign.exe` où on veut, il marche sans
  rien à côté. Ajouter une stack = déposer un fichier dans `templates/`, aucune
  ligne de code à toucher.
- **Un fichier par stack, lisible tel quel.** Les templates sont de vrais
  `.gitignore` commentés en français, pas des chaînes noyées dans le Go. On les
  relit, on les corrige, on les complète comme n'importe quel fichier.
- **On déduplique les motifs, pas les commentaires.** Node et Python ignorent
  tous deux `build/` : la règle n'apparaît qu'une fois, à sa première mention.
  Mais un commentaire `# Build` répété reste — c'est un repère de lecture, pas
  une règle, et le supprimer rendrait les sections bancales.
- **Un nom inconnu arrête tout.** `gign go cobl` ne va pas produire un fichier
  à moitié juste en silence : il s'arrête et renvoie vers `--list`. Mieux vaut
  une erreur franche qu'un `.gitignore` à trous qu'on ne remarque que trop tard.
- **Des alias pour les réflexes.** `golang`, `js`, `py`, `idea`, `osx`… tombent
  sur le bon template. On tape ce qui vient, pas le nom canonique.

## Ce que j'ai laissé tomber

- **Le catalogue complet de gitignore.io.** Une dizaine de stacks courantes
  suffit à 90 % des dépôts. Le reste, c'est un fichier à ajouter le jour où.
- **La fusion avec un `.gitignore` existant.** `-o` écrit le fichier, point. Pour
  compléter sans écraser, on garde la main : `gign go >> .gitignore`.
- **Les drapeaux après les stacks.** `flag` du stdlib s'arrête au premier mot
  qui n'est pas une option : `-o` se met donc avant les stacks. Convention de
  tout le repo, autant s'y tenir.
