package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	work := flag.String("work", "25", "durée d'une séance de focus")
	short := flag.String("short", "5", "durée d'une pause courte")
	long := flag.String("long", "15", "durée de la grande pause")
	rounds := flag.Int("rounds", 4, "nombre de séances avant la grande pause")
	loop := flag.Bool("loop", false, "enchaîner les cycles sans fin (Ctrl-C pour sortir)")
	flag.Usage = usage
	flag.Parse()

	plan, err := planFromArgs(*work, *short, *long, *rounds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pomo:", err)
		os.Exit(1)
	}

	// On capte Ctrl-C nous-mêmes pour rendre la main proprement : un retour à
	// la ligne après la barre, plutôt qu'une sortie en plein milieu du tracé.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	printPlan(plan, *loop)
	for {
		for i, p := range plan {
			runPhase(p, i+1, len(plan), sigs)
			title, body := message(plan, i, *loop)
			notify(title, body)
		}
		if !*loop {
			break
		}
	}
	fmt.Println("Terminé. Beau travail \U0001F44F")
}

// planFromArgs construit la liste des phases. Un argument positionnel seul
// (pomo 10m ) bascule en minuteur unique, sans pauses : pratique pour
// chronométrer un truc isolé sans monter tout un cycle.
func planFromArgs(work, short, long string, rounds int) ([]Phase, error) {
	switch flag.NArg() {
	case 0:
		w, err := parseDur(work)
		if err != nil {
			return nil, err
		}
		s, err := parseDur(short)
		if err != nil {
			return nil, err
		}
		l, err := parseDur(long)
		if err != nil {
			return nil, err
		}
		return buildPlan(w, s, l, rounds), nil
	case 1:
		d, err := parseDur(flag.Arg(0))
		if err != nil {
			return nil, err
		}
		return []Phase{{Focus, d}}, nil
	default:
		usage()
		os.Exit(2)
		return nil, nil 
	}
}

// runPhase décompte une phase seconde par seconde en réécrivant la même ligne.
// On vise une échéance absolue (time.Until) plutôt que de soustraire à chaque
func runPhase(p Phase, idx, total int, sigs <-chan os.Signal) {
	end := time.Now().Add(p.Dur)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	draw := func() {
		rem := time.Until(end)
		if rem < 0 {
			rem = 0
		}
		fmt.Printf("\r  %s  %-12s %s %s  [%d/%d]   ",
			icon(p.Name), p.Name, clock(rem), bar(p.Dur-rem, p.Dur), idx, total)
	}

	draw()
	for {
		select {
		case <-sigs:
			fmt.Println("\n  interrompu.")
			os.Exit(0)
		case <-tick.C:
			if time.Until(end) <= 0 {
				draw()
				fmt.Println()
				return
			}
			draw()
		}
	}
}

// message choisit le titre et le corps de la notification selon ce qui vient
// de finir et ce qui suit : on encourage à souffler après le focus, à reprendre
// après la pause, et on salue la fin du cycle.
func message(plan []Phase, i int, loop bool) (title, body string) {
	last := i == len(plan)-1
	if last && !loop {
		return "Pomo", "Cycle terminé \U0001F389 Beau travail !"
	}
	if plan[i].Name == Focus {
		return "Pomo — pause", "Séance terminée. Souffle un peu ☕"
	}
	return "Pomo — focus", "Pause finie. On s'y remet \U0001F345"
}

// icon donne un pictogramme par type de phase, histoire de repérer d'un coup
// d'œil si on bosse ou si on souffle.
func icon(name string) string {
	switch name {
	case Focus:
		return "\U0001F345"
	default:
		return "☕" 
	}
}

// printPlan résume le programme avant de lancer le premier décompte.
func printPlan(plan []Phase, loop bool) {
	var total time.Duration
	for _, p := range plan {
		total += p.Dur
	}
	suffix := ""
	if loop {
		suffix = " puis on recommence"
	}
	phase := "phases"
	if len(plan) == 1 {
		phase = "phase"
	}
	fmt.Printf("pomo — %d %s, %s au total%s. Ctrl-C pour arrêter.\n",
		len(plan), phase, clock(total), suffix)
}

func usage() {
	fmt.Fprint(os.Stderr, `pomo — minuteur Pomodoro en terminal

  pomo [options]        cycle complet (focus / pauses)
  pomo <durée>          minuteur unique, sans pauses

Options :
  -work  <durée>   séance de focus           (déf. 25)
  -short <durée>   pause courte              (déf. 5)
  -long  <durée>   grande pause finale       (déf. 15)
  -rounds <n>      séances avant la grande pause (déf. 4)
  -loop            enchaîner les cycles sans fin

Une durée est un nombre de minutes (« 25 ») ou une durée Go (« 90s », « 1h30m »).

Exemples :
  pomo                       cycle 25/5, grande pause de 15 après 4 séances
  pomo -work 50 -short 10    sessions plus longues
  pomo 10m                   un seul minuteur de 10 minutes
  pomo -loop                 tourner toute la journée
`)
}
