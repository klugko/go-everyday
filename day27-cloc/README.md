# day27 — cloc

« Ça fait combien de lignes, ce projet ? » Question banale, réponse pénible à la
main. `cloc` parcourt l'arbre, reconnaît les langages à l'extension, et sort un
tableau : code, commentaires, lignes vides, par langage, trié par poids.

```
cloc                # le dossier courant
cloc ../mon-projet  # ailleurs
```

```
--------------------------------------------------
Langage        Fichiers    Vides     Comm.     Code
--------------------------------------------------
Go                  84      983       883     8788
Markdown            28      424         0     1734
--------------------------------------------------
Total              112     1407       883    10522
--------------------------------------------------
```

## Décisions qui ont compté

- **Trois colonnes, pas une.** Le nombre de lignes brut ment : 200 lignes dont la
  moitié de commentaires, ce n'est pas 200 lignes de code. On sépare donc vides,
  commentaires et code — c'est cette dernière colonne qu'on regarde, donc c'est
  elle qui trie le tableau.
- **Code + commentaire sur la même ligne = code.** `x := 1 // pourquoi` compte
  pour du code, comme dans le vrai `cloc`. Compter cette ligne deux fois, ou la
  ranger en commentaire, fausserait le total. Règle simple, assumée.
- **Un suivi d'état pour les blocs.** Un `/* … */` sur plusieurs lignes met le
  compteur dans un mode « bloc » jusqu'à la fermeture. Le cas tordu — bloc ouvert
  jamais refermé — est testé : tout ce qui suit reste du commentaire, on ne
  retombe pas par erreur en code.
- **La table de langages se lit en un coup d'œil.** Une `map[extension]lang`, où
  un `lang` n'est que ses marqueurs de commentaire. Ajouter Kotlin ou PHP, c'est
  une ligne. Les familles `//` et `#` ont leur petit constructeur pour ne pas se
  répéter.
- **On ne compte que ce qu'on connaît.** Extension inconnue → fichier ignoré, pas
  rangé dans un fourre-tout « Autres » qui gonflerait le total sans rien dire.
  Comme [todoscan](../day26-todoscan), on saute aussi `.git`, `node_modules` et
  consorts.
- **Le saut de ligne final ne crée pas de ligne vide.** Détail, mais un fichier
  bien formé finit par `\n` ; le compter comme une ligne vide ajouterait un faux
  vide par fichier. On retire ce seul `\n` de queue, et les vraies lignes vides
  internes restent comptées.

## Ce que j'ai laissé tomber

- **La détection fine des commentaires imbriqués.** Rust autorise les `/* /* */ */`
  emboîtés ; on les traite comme un bloc plat. Cas rare, gain nul pour la
  complexité ajoutée.
- **Les chaînes contenant des marqueurs.** Une string `"// pas un commentaire"`
  en début de ligne serait comptée comme commentaire. Distinguer demanderait un
  vrai lexer par langage — disproportionné pour un compteur de lignes.
- **Le détail par fichier.** On agrège par langage, c'est ce qu'on veut 99 % du
  temps. Le détail fichier par fichier serait un mode de plus, pas la vue par
  défaut.
- **Le format machine (CSV/JSON).** Le tableau est fait pour l'œil. Une sortie
  exploitable par script, c'est l'évolution évidente — mais [csv2x](../day21-csv2x)
  et compagnie montrent que c'est un autre métier.
