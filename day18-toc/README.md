# day18 — toc

Insérer une table des matières dans un Markdown, et surtout **la garder à
jour** quand le document bouge. Pas un one-shot qu'on relance en effaçant tout
à la main : un outil qu'on repasse autant de fois qu'on veut sans rien casser.

## Le problème

Un README qui grossit finit par avoir besoin d'un sommaire. Le générer une
fois, c'est dix minutes. Le maintenir quand on ajoute trois sections et qu'on
en renomme deux, c'est l'enfer — et au final personne ne le fait, la table
ment au bout d'une semaine. Je voulais que régénérer soit gratuit : je relance,
ça se met à jour, je ne réfléchis plus.

```
toc README.md            # affiche le résultat sur stdout, pour voir
toc -w README.md         # réécrit le fichier sur place
cat doc.md | toc         # stdin marche aussi, en lecture seule
```

## Décisions qui ont compté

- **Deux marqueurs HTML, et l'idempotence vient toute seule.** La table vit
  entre `<!-- toc -->` et `<!-- /toc -->`. Tout ce qui est entre les deux
  m'appartient et part à la poubelle au passage suivant ; le reste du fichier
  n'est jamais touché. Du coup repasser l'outil dix fois donne exactement le
  même fichier — c'est ce que vérifie le test d'idempotence.
- **Pas de marqueurs ? la table se glisse après le premier titre.** Le cas du
  tout premier passage. On pose la table juste sous le `# Titre`, entourée de
  blancs, avec ses marqueurs — et dès lors les passages suivants n'ont plus
  qu'à remplacer l'entre-deux. Un marqueur d'ouverture posé seul quelque part
  sert aussi de point d'insertion, si on veut choisir l'endroit.
- **Des ancres façon GitHub.** Minuscules, espaces en tirets, ponctuation
  jetée. Et le détail qui pique : deux titres identiques se voient suffixer
  `-1`, `-2`… exactement comme GitHub numérote ses ancres, sinon les deux liens
  tomberaient au même endroit.
- **L'indentation est relative au titre le moins profond.** Une doc qui
  commence en `##` ne se retrouve pas décalée d'un cran pour rien : on prend le
  niveau le plus haut présent comme base.
- **On saute les blocs de code.** Un `# commentaire` dans un bloc ```` ``` ````
  shell n'est pas un titre de section. Une simple bascule sur les clôtures
  suffit, pas besoin d'un vrai parseur.
- **Le style de fin de ligne est préservé.** Un fichier en CRLF reste en CRLF.
  Sinon `-w` réécrirait tout le fichier d'un coup et le diff git serait
  illisible.

## Ce que j'ai laissé tomber

- **Choisir les niveaux à inclure** (`--min`, `--max`). J'inclus tous les
  titres et je laisse l'indentation relative faire le tri visuel. Un flag de
  plus pour un besoin que je n'ai pas eu encore.
- **Réécrire en lecture seule sur stdin.** `-w` exige un vrai fichier : il n'y
  a rien à réécrire dans un pipe. stdin reste un mode « je regarde ».
- **Les titres Setext** (soulignés par `===` ou `---`). Plus personne ne les
  écrit ; les `#` couvrent tout ce que je tape.
- **Nettoyer le libellé du lien.** Le texte du titre part tel quel dans la
  table, formatage Markdown compris — un `## Le *vrai* truc` reste en italique
  dans le sommaire. C'est cohérent, et ça m'évite un deuxième mini-parseur.
