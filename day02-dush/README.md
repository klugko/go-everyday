# dush

`du -sh` te dit "ce dossier fait 47 GiB" et après tu te retrouves à faire
`du -sh *` à chaque niveau pour trouver où c'est passé. Au bout du 5e
niveau t'as déjà perdu 30 secondes et la patience. L'idée bête : un truc
qui te crache direct le top N des plus gros trucs (fichiers ET dossiers),
à plat, peu importe où ils sont planqués dans l'arbre.

## Le cheminement

Premier truc à régler : la vitesse. Un `filepath.WalkDir` séquentiel sur
50 Go passe son temps à attendre les syscalls stat. Si on descend en
parallèle dans les sous-dossiers, on tape facile un 5–10x. C'est la
différence entre "ça marche" et "ça marche en 2 secondes".

Sauf qu'on peut pas balancer une goroutine par dossier  sur un disque
qui a 100k dossiers ça ferait 100k stacks, plusieurs centaines de Mo
pour rien. Idée : un sémaphore (channel bufferé taille `NumCPU*2`). Si
y'a un slot libre on spawn, sinon on continue inline dans le goroutine
courant. Le compte de goroutines reste plafonné, et la récursion fait
toujours avancer le boulot  pas de deadlock possible.

Pour la somme des tailles d'un dossier, j'ai d'abord mis un mutex, puis
je me suis dit qu'`atomic.Int64` était plus simple. Une ligne au lieu de
trois.

Deuxième truc : le top N. La version naïve c'est "je collecte tout, je
trie, je prends les N premiers". Sauf qu'un disque peut avoir des
millions de fichiers, et stocker 5M de (path, size) c'est ~400 Mo pour
rien. Min-heap de taille N : on garde que les N plus gros vus jusqu'ici.
Si le candidat suivant est plus petit que le min du heap, on jette.
O(M log N) au lieu de O(M log M), mémoire plafonnée à N. La vraie raison
c'est surtout la mémoire  le tri à la fin coûte trois fois rien à
côté.

`container/heap` de la stdlib est un peu verbeux (Len, Less, Swap, Push,
Pop à implémenter) mais ça tient en 5 lignes. Pas de quoi pleurer.

Troisième truc : faut compter les dossiers comme des entrées à part
entière. Sa taille = somme récursive de son contenu. Comme la récursion
remonte déjà les tailles, c'est gratuit  à la fin de `walk` on push
`(path, total)` dans le heap. Du coup le top mélange `node_modules/ (2 GiB)`
et `big.iso (4 GiB)` naturellement, et c'est exactement ce qu'on veut
pour "qu'est-ce que je peux supprimer".

Petit piège : la racine elle-même finit dans le heap parce qu'elle est
forcément la plus grosse. On la filtre à l'affichage, et on demande N+1
au heap pour avoir vraiment N résultats utiles. Deux lignes en plus, on
s'en remet.

## Ce que j'ai laissé tomber

- Suivi des symlinks. Boucle infinie possible, et de toute façon un
  symlink fait quelques octets  c'est jamais ce qu'on veut quand on
  cherche à libérer du disque.
- Filtres par type/âge/extension. C'est `du` qu'on construit, pas `find`.
- Mode "que les dossiers" ou "que les fichiers". Le mélange c'est
  précisément ce qui rend l'outil utile : tu vois en un coup d'œil si
  c'est un gros dossier ou un gros fichier qui te bouffe l'espace.

## Usage

```
go run . [-n N] [chemin]
```

Par défaut : `N=20`, chemin = `.`.

Exemple de sortie :

```
   4.2 GiB  Downloads/big.iso
   3.1 GiB  node_modules/
   1.8 GiB  .cache/
   ...
```

## Organisation

```
main.go   CLI, point d'entrée
dush.go   walk concurrent + heap top-N + formatage
```

