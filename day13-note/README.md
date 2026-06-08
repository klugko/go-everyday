# day13 — note

Jeter une pensée, une tâche, un truc à ne pas oublier — **sans casser le flux**.
Une commande, et c'est rangé dans le Markdown du jour, daté et horodaté.

## Le problème

Quand une idée passe, le coût d'entrée doit être proche de zéro : ouvrir un
fichier, chercher où écrire, retrouver la date du jour... à ce prix-là on ne
note rien. Je veux taper `note "..."` et que ça atterrisse au bon endroit tout
seul, pour relire la journée d'un coup le soir.

```
note acheter du café              # ajoute la note au fichier du jour
note "réunion repoussée à 16h"    # les guillemets si tu as des caractères spéciaux
echo "idée en passant" | note     # lit stdin, pratique en pipe
note --show                       # relit le fichier du jour
note --dir ~/journal ...          # range ailleurs
```

Le fichier ressemble à ça :

```markdown
# 2026-06-08

## 14:32

acheter du café

## 16:05

réunion repoussée à 16h
```

## Décisions qui ont compté

- **Un fichier par jour, nommé par sa date** (`2026-06-08.md`). Un `ls` du
  dossier *est* déjà le sommaire ; pas besoin d'index ni de base de données.
- **On append, jamais on réécrit.** `O_APPEND` : chaque note se colle à la fin,
  même si deux instances tournent. Le titre du jour ne s'écrit qu'une fois —
  on le déduit d'un fichier encore vide (`Size() == 0`), pas d'un test
  d'existence à part.
- **Le temps est un paramètre, pas un appel caché.** `appendNote`, `entry` et
  `header` reçoivent un `time.Time`. Du coup les tests figent une date et
  vérifient le format exact, sans horloge qui bouge sous les pieds.
- **Date en ISO `2006-01-02`.** Le format Go ne sait pas dire les jours en
  français, et l'ISO trie tout seul dans l'ordre chronologique. Plus simple,
  plus utile.

## Ce que j'ai laissé tomber

- **Tags / recherche.** Le fichier est du Markdown brut : `grep` fait déjà le
  job. Pas la peine de réinventer un moteur de recherche.
- **Édition / suppression de notes.** Si tu veux corriger, tu ouvres le `.md`.
  L'outil sert à *capturer* vite, pas à gérer.
- **Configuration.** Juste `--dir` et la variable `NOTE_DIR` (défaut `~/notes`).
  Rien d'autre à régler.
