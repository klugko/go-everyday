# day29 — commitlint

Les messages de commit dérivent vite : un coup `Fix bug`, un coup `update`, un
coup `feat:truc` sans espace. `commitlint` lit un message et dit, en clair, ce
qui ne colle pas à la convention `type(scope): description`. Et comme il sort en
erreur quand le message est mauvais, il se branche tel quel en hook `commit-msg`
pour refuser le commit avant qu'il parte.

```
commitlint .git/COMMIT_EDITMSG          # ce que fait un hook commit-msg
git log -1 --format=%B | commitlint     # vérifier le dernier commit
commitlint -make -type feat -scope day29 -desc "valide les messages"
```

## Décisions qui ont compté

- **Silencieux quand c'est bon.** Message conforme → code 0, rien sur la sortie.
  C'est exactement ce qu'un hook attend : il ne parle que pour bloquer. Les
  problèmes partent sur stderr, en liste à puces, un défaut par ligne.
- **Des erreurs qui apprennent, pas qui grondent.** Chaque problème dit *quoi*
  corriger (« il manque le « : » », « type inconnu — autorisés : … »), et la fin
  rappelle le format avec un exemple. C'est la moitié « aide à rédiger » du
  cahier des charges : le linter enseigne la convention au passage.
- **On lit la première ligne et on s'arrête là où ça compte.** Le sujet porte
  99 % de la convention : type, scope, `!` de breaking change, `: `, longueur. Le
  corps n'a qu'une règle — une ligne vide doit le séparer du sujet, sinon `git`
  fond les deux dans `%s`.
- **Le mode `-make` passe par le même linter.** Il assemble `type(scope): desc`
  puis le relit avant de l'imprimer : impossible de produire un sujet que l'outil
  refuserait. Pas de seconde implémentation des règles à maintenir.
- **Les commentaires de git sont ignorés.** `COMMIT_EDITMSG` est truffé de lignes
  `#` d'aide ; on les jette, on coupe les blancs du début, et le vrai sujet est ce
  qui reste en tête. Sinon le hook planterait sur son propre gabarit.
- **Les types sont une option, pas une liste gravée.** `-types` et `-max`
  laissent chaque projet poser ses règles. Le défaut suit la convention Angular
  (`feat, fix, docs, …`) parce que c'est ce que la plupart des dépôts attendent.

## Ce que j'ai laissé tomber

- **Un fichier de config.** Pas de `.commitlintrc` à parser : deux flags
  (`-types`, `-max`) couvrent ce qu'on règle vraiment, et un hook les écrit une
  fois pour toutes. Un fichier JSON en plus, ce serait du poids pour rien.
- **La casse de la description.** « commence par une minuscule », « à
  l'impératif »… des règles à faux positifs garantis dès qu'il y a un nom propre
  ou de l'anglais. Je m'en tiens au mesurable : pas de point final, pas vide.
- **Le parsing du footer.** `BREAKING CHANGE:`, `Refs #12`, les co-auteurs… on ne
  les valide pas. Le `!` dans le sujet suffit à marquer un breaking change, et le
  reste du corps est libre — le forcer apporterait surtout des refus pénibles.
- **Le mode interactif.** Pas de questions-réponses pour composer un message :
  `-make` prend des flags, scriptable et sans surprise. L'interactivité, c'est le
  boulot de l'éditeur de commit, pas d'un linter.
