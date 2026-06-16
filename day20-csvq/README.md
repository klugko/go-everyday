# day20 — csvq

Du SQL sur un CSV. `SELECT … WHERE … GROUP BY …`, en ligne de commande, sans
charger le fichier dans un tableur ni écrire un script Python jetable. Le genre
de question qu'on se pose dix fois par jour sur un export : « combien de lignes
par ville ? », « la moyenne des salaires au-dessus de 30 ans ? ».

## Le problème

Un CSV traîne dans un dossier, on veut juste *interroger* dedans. `grep` filtre
des lignes mais ne sait ni grouper ni faire une moyenne ; ouvrir Excel pour
trois lignes est ridicule. SQL est exactement le bon langage pour ça — il ne
manquait qu'un moteur minuscule qui le parle au-dessus d'un fichier plat.

```
csvq "SELECT name, age FROM gens.csv WHERE city = 'Paris' AND age > 28"
csvq "SELECT city, count(*) AS n, avg(salary) FROM gens.csv GROUP BY city ORDER BY n DESC"
csvq "SELECT name FROM gens.csv WHERE name LIKE 'a%' LIMIT 5"
cat gens.csv | csvq "SELECT * FROM - WHERE age >= 30"   # '-' = entrée standard
```

Le fichier interrogé est celui nommé après `FROM`. La sortie est du CSV, donc
elle se rebranche dans un autre `csvq` ou dans n'importe quel outil de la chaîne.

## Décisions qui ont compté

- **Tokeniser une fois, parser ensuite.** Premier réflexe : découper la requête
  à coups de `strings.Split` sur « WHERE », « GROUP BY »… Ça casse dès qu'une
  valeur contient le mot (`WHERE name = 'fish and chips'`). Un petit tokeniseur
  qui respecte les apostrophes règle le problème proprement, et le reste du
  parseur se contente de consommer des jetons clause par clause. Pas de vrai
  analyseur SQL, mais pas non plus de découpage fragile.
- **Le WHERE en forme normale disjonctive.** `a AND b OR c` devient `(a ET b)`
  **OU** `c` : un OU de groupes ET. C'est la précédence de SQL (le ET lie plus
  fort), et ça rend l'évaluation triviale — une ligne passe si **un** groupe
  passe entièrement. Pas de parenthèses, c'est la limite assumée.
- **Comparer en nombres quand les deux côtés sont chiffrés.** `age > 9` doit
  garder toutes les lignes, pas seulement celles qui commencent par un chiffre
  ≥ 9 en texte. On tente un `ParseFloat` des deux côtés ; si ça marche, on
  compare en nombres, sinon en chaînes. C'est ce que l'œil attend.
- **L'agrégat se déclenche tout seul.** Dès qu'il y a un `GROUP BY` *ou* une
  fonction (`count`, `sum`, `avg`, `min`, `max`), on bascule en mode regroupé.
  Sans `GROUP BY`, tout le fichier est un seul groupe — d'où le `count(*)`
  global qui marche sans cérémonie. Les colonnes brutes à côté d'un agrégat
  prennent la valeur de la première ligne du groupe : permissif, mais c'est
  exactement ce qu'on veut écrire (`SELECT city, count(*) … GROUP BY city`).
- **`LIKE` insensible à la casse.** En interrogation ad hoc on tape vite et mal.
  `%` et `_` sont traduits en regex (`.*` et `.`), le motif est ancré, et on
  ignore la casse. Plus pratique que l'inverse pour ce genre d'usage.
- **On tolère le BOM et les lignes inégales.** Un CSV sorti d'Excel commence
  souvent par un BOM collé au premier en-tête : on le retire. Et `FieldsPerRecord
  = -1` évite de planter sur une ligne mal formée — une cellule manquante se lit
  comme vide plutôt que comme une erreur fatale.

## Ce que j'ai laissé tomber

- **JOIN, sous-requêtes, fonctions imbriquées.** C'est un outil pour *un*
  fichier. Le jour où il faut joindre deux CSV, c'est une base de données qu'il
  faut, pas ce script.
- **Les parenthèses dans le WHERE.** La forme normale disjonctive couvre
  l'écrasante majorité des filtres réels. Gérer `(a OR b) AND c` voudrait dire un
  vrai parseur d'expressions à précédence — pas rentable ici.
- **`ORDER BY` sur un agrégat directement.** On trie sur les colonnes de sortie
  par leur nom ; pour ordonner sur un `count(*)`, on lui donne un alias
  (`count(*) AS n … ORDER BY n`). Une ligne de plus, et le code reste simple.
- **Typage des colonnes.** Tout est texte ; le numérique est décidé à la volée,
  cellule par cellule. Pas de schéma, pas de déclaration — un CSV n'en a pas.
- **`DISTINCT`, `HAVING`, les types de date.** Au-delà du besoin du jour. `HAVING`
  serait le prochain ajout naturel (filtrer *après* le GROUP BY) si ça manquait.
