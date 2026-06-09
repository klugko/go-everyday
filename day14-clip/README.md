# day14 — clip

Un historique de presse-papiers. Tu as copié une URL, puis trois autres choses
par-dessus, et là tu la veux à nouveau : `clip` te ressort ce que tu avais sous
la main il y a dix minutes.

## Le problème

Le presse-papiers ne retient qu'**une** chose : la dernière. Dès qu'on recopie,
le reste est perdu. Or on copie en rafale toute la journée — un bout de code, un
lien, un numéro — et c'est presque toujours l'avant-dernier qu'on regrette.
Je veux deux gestes : un mouchard qui tourne en fond et note chaque copie, et
une commande qui me rend l'historique pour piocher dedans.

```
clip --watch                  # à lancer une fois, tourne en fond et enregistre
clip                          # relit l'historique, le plus récent en haut
clip --get 2                  # recopie la 2e entrée dans le presse-papiers
clip --clear                  # repart de zéro
clip --watch --every 2s       # scrute moins souvent
clip --dir D:\clip ...        # range l'historique ailleurs
```

La liste ressemble à ça :

```
  1  il y a 2 min   https://exemple.fr/page
  2  il y a 10 min  rendez-vous à 16h
  3  il y a 1 h      func humanAge(now, then time.Time) string { …
```

## Décisions qui ont compté

- **Deux rôles, un seul binaire.** `--watch` est le démon qui scrute ; sans
  drapeau, on affiche. Pas de service à installer : on lance `clip --watch` dans
  un terminal (ou au démarrage) et on oublie.
- **On scrute, on ne s'accroche pas au système.** Pas d'API native de
  surveillance du presse-papiers (différente sur chaque OS, fragile) : un simple
  `time.Sleep` toutes les secondes et on compare au dernier contenu vu. Une
  seconde de latence ne se sent pas, et le code reste lisible et portable.
- **Le presse-papiers via l'outil natif.** `Get-Clipboard` / `Set-Clipboard`
  sous Windows, `pbpaste` / `pbcopy` sous macOS, `xclip` sous Linux — lancés en
  sous-processus comme les notifications du day10. Zéro dépendance à compiler.
- **JSON, une entrée par ligne.** Une copie contient souvent des retours à la
  ligne ; le JSON les échappe, donc *une ligne = une entrée* reste vrai et
  `load` reste un simple scan ligne par ligne. Une ligne abîmée est sautée, pas
  fatale.
- **Plafond à 200 entrées.** Un mouchard qui tourne des jours remplirait le
  disque sinon. Au-delà, on rogne le plus vieux. Et un doublon collé juste
  après le même contenu est ignoré — recopier deux fois ne pollue pas la liste.
- **L'heure est un paramètre.** `record`, `humanAge` et `formatList` reçoivent
  un `time.Time`, donc les tests figent une date et vérifient l'âge au mot près.

## Ce que j'ai laissé tomber

- **Les images et fichiers copiés.** On ne garde que le texte. Le reste demande
  des formats par OS pour un usage bien plus rare.
- **La recherche.** L'historique est du JSON lisible : `grep` sur le fichier
  fait le job le jour où il en faut une.
- **Le chiffrement.** On peut copier un mot de passe sans le vouloir — mais
  chiffrer demanderait une clé à gérer. À la place : `--clear` sous la main, et
  le plafond qui finit par évacuer les vieilles entrées.
