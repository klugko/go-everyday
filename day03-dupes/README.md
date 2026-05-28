# dupes

Tu finis toujours par accumuler les mêmes fichiers en double : la photo
téléchargée trois fois, le PDF qu'on s'envoie à soi-même par mail, les
backups de backups. `dupes` parcourt un dossier, repère les vrais
doublons (même contenu, pas juste même nom), et te propose de les
supprimer un groupe à la fois.

## Le cheminement

Première question : c'est quoi un "doublon" ? Pas juste un fichier qui
porte le même nom — deux `IMG_2042.jpg` peuvent contenir des photos
totalement différentes. C'est le **contenu** qui compte. Donc :
hash cryptographique, et deux fichiers identiques ssi leurs hashes le
sont. SHA-256, stdlib, fin du débat.

Sauf qu'hasher chaque fichier d'un disque, c'est lire chaque octet d'un
disque. Sur 200 Go ça prend plusieurs minutes même en SSD. La bonne
nouvelle : deux fichiers de tailles différentes ne peuvent pas être
identiques. Et la taille, on l'a quasi gratuitement via `os.ReadDir` +
`DirEntry.Info()` — un syscall stat, pas une lecture de contenu.

D'où le pipeline :

1. **Scan** : on descend récursivement et on groupe les chemins par
   taille. Coût : quasiment celui d'un `find`.
2. **Hash** : pour chaque groupe de taille ≥ 2, on hashe les candidats.
   Les fichiers seuls de leur taille sont sautés direct — c'est la
   grosse économie. Sur un disque "normal" la plupart des tailles sont
   uniques, donc on touche peu de contenu.
3. **Groupage** : on regroupe par hash, on jette les groupes de taille 1
   (collision sur la taille mais contenu différent — l'imposteur).

Le hashing est parallélisé : N workers (= `NumCPU`) lisent les
candidats sur un channel `jobs` et publient les résultats sur un channel
`out`. C'est la partie qui bouffe du temps, c'est elle qui mérite la
parallélisation. Le scan, lui, je l'ai laissé séquentiel — un `ReadDir`
récursif suffit, le bottleneck c'est pas là.

Petit piège  : `cands := append(cands, ...)` puis une
seule goroutine "producteur" qui pousse dans `jobs`. J'ai hésité à
fusionner producteur et collecteur, mais avoir trois rôles séparés
(producteur / workers / consommateur) c'est trois fois 5 lignes très
lisibles. Pas la peine de tordre.

Pour la suppression, deux principes :

- **Jamais d'action par défaut.** Sans `-i`, l'outil ne supprime rien,
  il affiche. Il y a trop de scénarios où "ah mais celui-là c'était la
  version originale" — l'utilisateur doit décider.
- **On demande "lequel garder"**, pas "lequel supprimer". Avec "lequel
  garder", taper rien = on saute le groupe (action neutre). Avec "lequel
  supprimer", taper rien serait ambigu. Et un mauvais clic supprime au
  pire `N-1` fichiers, au lieu de risquer de supprimer le dernier
  original.

Les groupes sont triés par espace récupérable décroissant — `size ×
(copies − 1)`. Les gros groupes en premier, c'est ce qu'on veut traiter
en priorité. Ça change tout sur un dossier avec 2000 groupes de
doublons : les 10 premiers récupèrent souvent 80% de la place.

## Ce que j'ai laissé tomber

- **Hash partiel** (lire seulement les premiers KiB d'abord pour
  filtrer, puis hash complet si collision). Gain mesurable sur des
  fichiers très gros qui partagent un préfixe, mais le code double et
  pour mon usage le filtrage par taille suffit déjà largement.
- **Fichiers vides.** Tous identiques entre eux, jamais ce qu'on veut
  voir dans une liste de "doublons à nettoyer". On les saute.
- **Symlinks.** Risque de boucle, et "dédoublonner un alias" n'a aucun
  sens.
- **Mode auto "keep-first"** (supprimer sans demander, garder le premier
  par ordre alpha). Tentant pour scripter, mais l'irréversibilité me
  faisait flipper. Si je veux scripter, je piperai la sortie texte.

## Usage

```
go run . [chemin]      # liste les doublons, ne touche à rien
go run . -i [chemin]   # interactif : propose la suppression
```

Exemple de sortie :

```
[1] 4.2 MiB par fichier, 3 copies — 8.4 MiB récupérable
  1) photos/2024/IMG_2042.jpg
  2) photos/2024-backup/IMG_2042.jpg
  3) Downloads/IMG_2042.jpg

[2] 1.1 MiB par fichier, 2 copies — 1.1 MiB récupérable
  ...

Total : 9.5 MiB récupérable sur 2 groupes.
```

En mode `-i`, chaque groupe te demande :

```
Garder lequel ? (numéro / entrée pour passer / q pour quitter)
```

## Organisation

```
main.go        CLI + boucle interactive
dupes.go       scan, hash, group, tri, formatage
```
