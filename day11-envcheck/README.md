# envcheck

Tout le monde a déjà perdu une demi-heure sur une appli qui plante au
démarrage parce qu'une clé manquait dans le `.env`. Le `.env.example` est
là pour ça — c'est le contrat — mais personne ne le relit ligne à ligne.
`envcheck` fait la diff à ta place : il compare ton `.env` au modèle et
te dit ce qui **manque** et ce qui est **en trop**.

```
envcheck                                  # .env contre .env.example
envcheck -env .env.local -example .env.example
```

## Le cheminement

### On compare des clés, pas des valeurs

Le réflexe serait de parser proprement chaque ligne en `clé = valeur`.
Mais pour répondre à la question posée — « qu'est-ce qui manque ? » — la
valeur ne sert à rien, et elle est même un piège : un secret réel n'a
rien à faire dans une comparaison qu'on affiche à l'écran. Donc je ne
retiens que le **nom** de chaque variable. Le parsing se réduit à
« trouve le `=`, prends ce qu'il y a à gauche ».

### Ce qu'une ligne de `.env` peut cacher

Un `.env` n'est pas du tout structuré, alors le lecteur reste tolérant :
les lignes vides et les `# commentaires` sautent, le préfixe `export `
(habitude shell) est retiré, et une clé qui apparaît deux fois n'est
comptée qu'une fois. À l'inverse, je jette ce qui ne ressemble pas à une
affectation : une ligne sans `=`, ou un « nom » contenant un espace — du
texte collé par erreur plutôt qu'une vraie variable. Mieux vaut ignorer
un cas douteux que de le faire remonter comme un faux « en trop ».

### Garder l'ordre des fichiers

Les manquantes sont listées dans l'ordre du `.example`, les surnuméraires
dans l'ordre du `.env`. Trier alphabétiquement aurait été plus court, mais
on lit le diff bien plus vite quand il suit l'ordre du fichier qu'on a
sous les yeux — on retrouve la clé là où on s'attend à la voir.

### Sortir avec le bon code

Si quelque chose cloche, le programme rend un **code de sortie non nul**.
C'est ce qui le rend utile au-delà du terminal : une étape de CI ou un
hook de pré-commit peut planter le build quand un `.env.example` a gagné
une clé que personne n'a reportée. Un outil de vérification qui répond
toujours « tout va bien » à la machine ne sert pas à grand-chose.

## Ce que j'ai laissé tomber

- **Vérifier les valeurs.** Dire « cette clé est vide » ou « ce port
  n'est pas un nombre », c'est de la validation de config, un autre
  métier. Ici on s'en tient à la présence des clés.
- **L'interpolation `${AUTRE_VAR}`** et les guillemets multi-lignes. Des
  raffinements de format `.env` que les vrais loaders gèrent ; pour
  comparer des noms de clés, ils n'apportent rien.
- **Réécrire le `.env` automatiquement.** Compléter les clés manquantes
  serait pratique, mais ça veut dire inventer des valeurs — exactement ce
  qu'un humain doit décider. L'outil signale, il ne devine pas.

## Usage

```
envcheck [options]
```

Options :

```
-env <chemin>       fichier à vérifier        (déf. ".env")
-example <chemin>   fichier modèle de référence (déf. ".env.example")
```

Codes de sortie : `0` tout concorde · `1` des clés diffèrent · `2` un
fichier est illisible.

## Organisation

```
main.go       CLI : lecture des deux fichiers, affichage, code de sortie
envcheck.go   Keys (lecture des clés) + Compare (le diff)
```
