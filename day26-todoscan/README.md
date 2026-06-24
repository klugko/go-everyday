# day26 — todoscan

Les `// TODO` s'accumulent et se perdent. `todoscan` parcourt tout le projet et
les ressort en `fichier:ligne`, format que tous les éditeurs savent ouvrir d'un
clic. Et comme il sort en code d'erreur quand il en reste, il a sa place dans un
hook de commit ou une CI : « pas de FIXME en production ».

```
todoscan                          # le dossier courant
todoscan ./src                    # un sous-dossier
todoscan -tags TODO,FIXME,HACK    # ses propres marqueurs
```

## Décisions qui ont compté

- **`fichier:ligne:` d'abord.** C'est le format universel des compilateurs et des
  greps : VS Code, vim, IntelliJ le rendent cliquable. Tout le reste de la sortie
  est cosmétique ; cette colonne-là est le contrat.
- **Sortie en erreur s'il reste des marqueurs.** Zéro TODO → code 0, sinon code 1.
  Ça transforme l'outil en garde-fou : `todoscan && git commit` refuse de passer
  tant qu'il traîne un FIXME. Le décompte part sur stderr, la liste sur stdout —
  on peut donc piper l'une sans polluer l'autre.
- **Frontière de mot autour des tags.** `\bTODO\b` matche le marqueur mais pas
  `TODOLIST` ni `mastodonte`. Sans ça, la moindre variable bien nommée déclenche
  un faux positif. Les tags sont passés par `regexp.QuoteMeta` au cas où l'un
  contienne un caractère spécial.
- **On saute les binaires, pas en devinant l'extension.** Un octet nul dans les
  512 premiers octets (lus en `Peek`, sans consommer le flux) = fichier non
  textuel. Plus fiable qu'une liste d'extensions, qui rate toujours un cas.
- **On ne fouille pas les nids à dépendances.** `.git`, `node_modules`, `vendor`,
  `dist`, et tout dossier caché sont coupés à la racine via `SkipDir` : leurs
  TODO ne sont pas les tiens, et c'est 90 % du temps de scan en moins.
- **Un fichier illisible n'arrête pas le scan.** Permission refusée sur un
  fichier ? On le signale sur stderr et on continue. Un scan qui s'arrête au
  premier caillou ne sert à rien.

## Ce que j'ai laissé tomber

- **L'analyse syntaxique des commentaires.** On matche le tag où qu'il soit, pas
  seulement dans un vrai commentaire. Un `TODO` dans une chaîne de caractères
  remonte aussi — mais en pratique c'est presque toujours un vrai TODO, et parser
  chaque langage pour distinguer code et commentaire serait disproportionné.
- **Le respect du `.gitignore`.** [day15-gign](../day15-gign) sait le lire ; ici
  la liste en dur (`node_modules`, dossiers cachés…) couvre l'essentiel sans
  traîner cette logique. Si le besoin se confirme, ce serait l'évolution naturelle.
- **Le regroupement par fichier ou par auteur.** Une ligne plate par marqueur,
  triable et grepable. Regrouper, c'est le boulot d'un `| sort` ou d'un `| awk`
  derrière, pas de l'outil.
- **La priorité (TODO vs FIXME).** Tous les tags sont traités à égalité. Vouloir
  faire échouer la CI sur FIXME mais pas sur TODO, c'est deux passes avec deux
  `-tags` différents — pas un niveau de sévérité à coder.
