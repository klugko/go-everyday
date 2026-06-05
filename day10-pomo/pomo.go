package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Phase est un segment du minuteur : un nom à afficher et une durée.
type Phase struct {
	Name string
	Dur  time.Duration
}

// Les trois temps de la méthode Pomodoro. On les garde en constantes de nom
// pour que l'affichage et les messages de notification décident quoi dire
// sans comparer des chaînes en dur un peu partout.
const (
	Focus     = "Focus"
	Pause     = "Pause"
	LongPause = "Grande pause"
)

// buildPlan déroule un cycle complet : N séances de focus séparées par des
// pauses courtes, et une grande pause à la toute fin pour récompenser la
// série. C'est l'ossature classique de Pomodoro, sans surprise volontaire.
func buildPlan(work, short, long time.Duration, rounds int) []Phase {
	if rounds < 1 {
		rounds = 1
	}
	plan := make([]Phase, 0, rounds*2)
	for i := 1; i <= rounds; i++ {
		plan = append(plan, Phase{Focus, work})
		if i == rounds {
			plan = append(plan, Phase{LongPause, long})
		} else {
			plan = append(plan, Phase{Pause, short})
		}
	}
	return plan
}

// parseDur accepte deux écritures pour rester confortable : un nombre nu vaut
// des minutes (« 25 » = 25 min, le réflexe Pomodoro), sinon on tombe sur la
// syntaxe Go (« 90s », « 1h30m »). Une durée nulle ou négative n'a pas de sens
// pour un minuteur, donc on la refuse.
func parseDur(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("durée positive attendue : %q", s)
		}
		return time.Duration(n) * time.Minute, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("durée invalide : %q", s)
	}
	return d, nil
}

// clock formate un reste en MM:SS, et bascule en H:MM:SS dès qu'on dépasse
// l'heure — inutile d'afficher des heures à zéro le reste du temps.
func clock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// bar dessine une barre qui se remplit à mesure que la phase avance : vide au
// départ, pleine à l'échéance. Donne le ressenti du temps écoulé d'un coup d'œil.
func bar(elapsed, total time.Duration) string {
	const width = 24
	filled := 0
	if total > 0 {
		filled = int(float64(width) * elapsed.Seconds() / total.Seconds())
	}
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}
