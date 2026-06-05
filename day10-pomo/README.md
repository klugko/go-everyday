# pomo

Un minuteur Pomodoro, c'est presque trop simple pour mériter du code : on
attend, ça sonne. Sauf que « ça sonne » dans un terminal, c'est rarement
suffisant — on a la tête ailleurs, dans une autre fenêtre. Le vrai sujet de
ce jour, ce n'est pas le décompte, c'est la **notification système** : faire
surgir une bulle même quand le terminal est minimisé, sans rien installer.

## Le cheminement

### Le décompte : viser l'heure, pas compter les ticks

Premier réflexe : « toutes les secondes, j'enlève une seconde au compteur ».
Mauvaise idée — `time.Sleep` et les tickers ne sont jamais pile à l'heure, et
sur une séance de 25 minutes les petits retards s'additionnent. Alors je fixe
une **échéance absolue** au début de la phase (`time.Now().Add(durée)`) et,
à chaque tick, j'affiche `time.Until(échéance)`. Le ticker peut tressauter,
le minuteur reste juste : il se termine à la bonne seconde, point.

### Réécrire la ligne plutôt que la dérouler

Comme le `-watch` de `day08`, le décompte se réimprime sur place avec `\r` :
une seule ligne vivante avec le temps restant et une barre qui se remplit.
La barre se remplit (au lieu de se vider) parce que voir la progression
*avancer* est plus motivant que regarder une réserve fondre — détail, mais
c'est le genre d'outil qu'on fixe pendant 25 minutes.

### La notification, trois mondes trois façons

C'est là que ça se corse, et c'est tout l'intérêt. Pas de dépendance
externe : on appelle l'outil natif de chaque OS.

- **Windows** (cible principale) : une `NotifyIcon` avec bulle d'info, pilotée
  par un petit script PowerShell. C'est présent sur tout Windows récent sans
  module à installer, et le système la transforme en toast sur 10/11. Le
  script doit *dormir* le temps que la bulle vive ses quelques secondes, puis
  se nettoyer — d'où le `Start-Sleep` avant `Dispose`.
- **macOS** : `osascript -e 'display notification …'`, le réflexe AppleScript.
- **Linux** : `notify-send`, la convention freedesktop.

Deux pièges m'ont occupé. D'abord, **ne pas attendre** la commande : si le
minuteur se bloquait six secondes le temps du toast, l'enchaînement focus →
pause prendrait du retard. Donc `Start()` sans `Wait()` synchrone, et on
récupère le processus dans une goroutine. Ensuite, **l'échappement** : un
titre qui contiendrait une apostrophe casserait le script PowerShell ; je
double les apostrophes (et j'échappe les guillemets côté AppleScript) pour
glisser n'importe quel texte sans rien casser.

Et au-dessus de tout ça, la **cloche du terminal** (`\a`), envoyée dans tous
les cas. Pas de bureau graphique, SSH, console nue ? On entend quand même la
fin. La notification visuelle est un bonus, jamais le seul canal.

### Des durées qu'on écrit comme on pense

On dit « une séance de 25 » sans préciser « minutes ». Donc un nombre nu vaut
des minutes (`pomo -work 25`), et si on veut être précis la syntaxe Go reste
ouverte (`90s`, `1h30m`). Tester trois secondes pendant le développement
(`pomo 3s`) ne demande pas un mode spécial : c'est la même porte d'entrée.

## Ce que j'ai laissé tomber

- **Mettre en pause / sauter une phase au clavier.** Lire `stdin` en parallèle
  du décompte (touche `p` pour pause, `s` pour passer) doublait la complexité
  pour un minuteur qu'on lance et qu'on laisse tourner. Ctrl-C suffit, et il
  rend la main proprement (retour à la ligne après la barre, pas de sortie en
  plein tracé).
- **Un journal des sessions** (« 4 pomodoros faits aujourd'hui »). Utile, mais
  c'est un autre projet : il faut un fichier d'état, une notion de jour, des
  stats. Ici on reste sur le minuteur pur.
- **Choisir le son de la notification.** Le défaut système fait le travail ;
  paramétrer ça par OS aurait gonflé le code sans grand bénéfice.

## Usage

```
pomo [options]        cycle complet (focus / pauses)
pomo <durée>          minuteur unique, sans pauses
```

Options :

```
-work  <durée>   séance de focus                 (déf. 25)
-short <durée>   pause courte                    (déf. 5)
-long  <durée>   grande pause finale             (déf. 15)
-rounds <n>      séances avant la grande pause   (déf. 4)
-loop            enchaîner les cycles sans fin
```

Une durée est un nombre de minutes (« 25 ») ou une durée Go (« 90s », « 1h30m »).

Exemples :

```
pomo                       cycle 25/5, grande pause de 15 après 4 séances
pomo -work 50 -short 10    sessions plus longues
pomo 10m                   un seul minuteur de 10 minutes
pomo -loop                 tourner toute la journée
```

## Organisation

```
main.go     CLI, construction du cycle, boucle de décompte et affichage
pomo.go     phases, parsing des durées, horloge et barre de progression
notify.go   notification système par OS + cloche du terminal
```
