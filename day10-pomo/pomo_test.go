package main

import (
	"testing"
	"time"
)

func TestParseDur(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"25", 25 * time.Minute, false},   
		{" 5 ", 5 * time.Minute, false},  
		{"90s", 90 * time.Second, false},  
		{"1h30m", 90 * time.Minute, false},
		{"1.5m", 90 * time.Second, false},
		{"0", 0, true},   
		{"-5", 0, true},  
		{"abc", 0, true}, 
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseDur(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDur(%q) : erreur attendue, obtenu %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDur(%q) : erreur inattendue %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseDur(%q) = %v, voulu %v", c.in, got, c.want)
		}
	}
}

func TestBuildPlan(t *testing.T) {
	work, short, long := 25*time.Minute, 5*time.Minute, 15*time.Minute
	plan := buildPlan(work, short, long, 4)

	if len(plan) != 8 {
		t.Fatalf("4 rounds = 8 phases, obtenu %d", len(plan))
	}
	// Alternance focus / pause, avec la grande pause en dernier.
	for i := 0; i < len(plan); i += 2 {
		if plan[i].Name != Focus || plan[i].Dur != work {
			t.Errorf("phase %d : focus attendu, obtenu %+v", i, plan[i])
		}
	}
	if plan[1].Name != Pause || plan[1].Dur != short {
		t.Errorf("phase 1 : pause courte attendue, obtenu %+v", plan[1])
	}
	last := plan[len(plan)-1]
	if last.Name != LongPause || last.Dur != long {
		t.Errorf("dernière phase : grande pause attendue, obtenu %+v", last)
	}
}

func TestBuildPlanRoundsFloor(t *testing.T) {
	// 0 ou moins est ramené à un seul round : focus + grande pause.
	plan := buildPlan(time.Minute, time.Minute, time.Minute, 0)
	if len(plan) != 2 || plan[0].Name != Focus || plan[1].Name != LongPause {
		t.Errorf("rounds=0 devrait donner [Focus, Grande pause], obtenu %+v", plan)
	}
}

func TestClock(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00"},
		{59 * time.Second, "00:59"},
		{25 * time.Minute, "25:00"},
		{90 * time.Minute, "1:30:00"},
		{-time.Second, "00:00"},
	}
	for _, c := range cases {
		if got := clock(c.d); got != c.want {
			t.Errorf("clock(%v) = %q, voulu %q", c.d, got, c.want)
		}
	}
}

func TestBar(t *testing.T) {
	total := 10 * time.Minute
	if got := bar(0, total); got != "["+repeat('-', 24)+"]" {
		t.Errorf("barre au départ = %q", got)
	}
	if got := bar(total, total); got != "["+repeat('#', 24)+"]" {
		t.Errorf("barre à la fin = %q", got)
	}
	// À mi-parcours, moitié pleine.
	mid := bar(5*time.Minute, total)
	if mid != "["+repeat('#', 12)+repeat('-', 12)+"]" {
		t.Errorf("barre à mi-parcours = %q", mid)
	}
	// Au-delà de la durée, on ne déborde pas.
	if got := bar(2*total, total); got != "["+repeat('#', 24)+"]" {
		t.Errorf("barre saturée = %q", got)
	}
}

func repeat(c rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}
