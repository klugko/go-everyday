# day21 — csv2x

Un CSV, c'est parfait pour la machine et pénible pour tout le reste. `csv2x` le
recrache dans le format dont on a besoin sur le moment : du **JSON** pour une
API ou un script, une **table Markdown** à coller dans une issue ou un doc, ou
un vrai **classeur Excel** (`.xlsx`) pour les gens qui vivent dans un tableur.

```
csv2x gens.csv                       # JSON sur la sortie standard
csv2x -f md gens.csv                 # table Markdown
csv2x -o gens.xlsx gens.csv          # Excel (format déduit de l'extension)
cat gens.csv | csv2x -f json -        # '-' = entrée standard
```

Le format vient du `-f`, ou se devine de l'extension du `-o` quand on en donne
un. Sans `-o`, ça sort sur le terminal — sauf pour le `.xlsx`, qui est binaire
et réclame donc un fichier.

## Décisions qui ont compté

- **Le JSON devine les types.** Un CSV ne connaît que du texte ; si on le
  recopiait tel quel, tout serait entre guillemets et le JSON ne servirait à
  rien. Donc `30` devient un nombre, `true`/`false` des booléens, une cellule
  vide un `null`. Mais `007` et `75001` posés sur un code postal restent du
  texte : un identifiant qui commence par zéro n'est pas une quantité, et le
  transformer en `7` serait une perte de données silencieuse. D'où le garde-fou
  sur les zéros de tête et le `+`.
- **Le JSON est écrit à la main.** Tentant de remplir une `map[string]any` et de
  la donner à `encoding/json` — sauf qu'une map Go se sérialise par clé triée,
  ce qui mélangerait l'ordre des colonnes. On parcourt donc l'en-tête dans
  l'ordre et on n'emprunte `json.Marshal` que pour échapper proprement chaque
  clé et chaque chaîne.
- **Le `.xlsx` est fabriqué à la main, sans dépendance.** Un fichier Excel n'est
  qu'une archive zip de quelques fichiers XML (le format Office Open XML).
  `archive/zip` + `encoding/xml` suffisent : quatre parties fixes (les
  relations, le classeur) et une feuille générée. Pas de bibliothèque tierce,
  fidèle à la règle « stdlib seulement » du dépôt.
- **Les chaînes Excel sont posées *inline*.** La voie classique passe par une
  table partagée (`sharedStrings.xml`) pour dédupliquer le texte. C'est un
  fichier et un niveau d'indirection de plus pour un gain nul à notre échelle :
  on écrit chaque chaîne dans sa cellule (`t="inlineStr"`). Un export, pas un
  moteur de tableur.
- **Markdown : on échappe ce qui casse le tableau.** Un `|` dans une cellule
  romprait l'alignement des colonnes, un saut de ligne aussi. On échappe le
  premier (`\|`) et on aplatit le second en espace. Et chaque ligne est
  recalée sur la largeur de l'en-tête, pour qu'une cellule manquante laisse un
  trou plutôt que de décaler tout le reste.
- **On tolère le BOM et les lignes inégales.** Comme les autres jours qui lisent
  du CSV : un BOM Excel sur le premier en-tête est retiré, et `FieldsPerRecord
  = -1` laisse passer une ligne trop courte (cellule lue comme vide) au lieu de
  planter.

## Ce que j'ai laissé tomber

- **Les styles, formats et formules Excel.** Gras sur l'en-tête, colonnes
  larges, types de date… c'est un export brut, pas une mise en page. Le `.xlsx`
  généré est volontairement minimal — juste assez pour qu'Excel et LibreOffice
  l'ouvrent sans broncher.
- **La déduplication des chaînes (`sharedStrings`).** Voir plus haut : pas
  rentable pour un export ponctuel.
- **YAML, TOML, HTML…** Trois formats couvrent l'écrasante majorité des besoins.
  Le jour où il en faut un quatrième, c'est un `case` de plus dans le dispatch,
  pas une refonte.
- **L'inférence de schéma sur toute la colonne.** Le type est décidé cellule par
  cellule, pas en regardant la colonne entière. Plus simple, et ça colle à la
  réalité d'un CSV où une colonne « numérique » cache parfois un `N/A`.
