# day16 — read

Coller un texte et savoir en un coup d'œil : **est-ce que ça se lit bien ?**
Nombre de mots, temps de lecture, et un score de lisibilité.

## Le problème

On écrit un mail, un README, un message un peu long — et on n'a aucune idée si
c'est digeste ou si on noie le lecteur sous des phrases à rallonge. Je voulais
un outil qui prenne un texte sur stdin ou un fichier et me sorte trois chiffres
qui parlent, sans config ni service en ligne.

```
read article.md                    # analyse un fichier
cat brouillon.txt | read           # ou lit stdin, pratique en pipe
read --wpm 150 article.md          # ajuste la vitesse de lecture
```

Ça donne :

```
mots            240
phrases         18
mots/phrase     13.3
temps de lecture 2 min
lisibilité      62/100 (moyen)
```

## Décisions qui ont compté

- **Score de Flesch (Reading Ease).** La formule classique : elle pénalise les
  phrases longues (`mots/phrase`) et les mots à rallonge (`syllabes/mot`). Plus
  c'est haut, plus c'est facile. On la traduit en mot du quotidien — « facile »,
  « difficile » — parce qu'un nombre brut ne parle à personne.
- **Les syllabes, comptées par groupes de voyelles.** Pas de dictionnaire
  phonétique : on compte les groupes de voyelles consécutives (`bateau` →
  `ba-teau` → 2). C'est grossier mais ça suffit largement pour situer un texte.
  Le 'e' muet final fait surcompter (`rythme` → 2), c'est la limite assumée.
- **Les phrases, comptées en balayant le texte, pas le mot.** En français le
  point d'exclamation est détaché : `aujourd'hui !`. Si on cherchait la
  ponctuation collée au dernier mot, on raterait la phrase. On compte donc les
  groupes de terminateurs (`.`, `!`, `?`, `…`) dans le texte entier — et `?!`
  ou `...` ne valent qu'une seule fin.
- **Tout se déduit de trois comptages.** `analyze` ne remplit qu'un struct
  `{mots, phrases, syllabes}`. Le score, le temps, les ratios en découlent —
  faciles à tester en figeant les comptages.
- **Temps arrondi à la minute supérieure, jamais zéro.** Un texte d'un mot
  prend « 1 min » : annoncer « 0 min » serait absurde.

## Ce que j'ai laissé tomber

- **Les variantes françaises du score** (Kandel-Moles et compagnie). Flesch est
  connu, lisible, et l'ordre des textes (du plus simple au plus dense) reste
  juste même si la valeur absolue est calée sur l'anglais.
- **Un vrai découpage syllabique.** Il faudrait un dictionnaire ou des règles
  phonétiques par langue. Hors sujet pour un indicateur qu'on lit en passant.
- **Détection de la langue, statistiques par paragraphe, export.** L'outil
  répond à une question simple ; `grep` et `wc` sont là pour le reste.
