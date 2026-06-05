package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// notify signale la fin d'une phase de deux façons complémentaires : la cloche
// du terminal (\a), qui marche partout même sans bureau graphique, et une vraie
// notification système quand l'OS sait en afficher une. On lance la commande
// sans l'attendre : le minuteur ne doit pas se figer le temps qu'un toast vive
// ses quelques secondes à l'écran.
func notify(title, body string) {
	fmt.Print("\a")

	cmd := notifyCmd(title, body)
	if cmd == nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return 
	}
	go cmd.Wait() 
}

// notifyCmd choisit l'outil natif selon l'OS. Chacun a sa façon de faire
// surgir une bulle ; on ne dépend d'aucune brique externe à installer.
func notifyCmd(title, body string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		// Une NotifyIcon avec bulle d'information : disponible sur tout
		// Windows  10/11.
		// Le script dort le temps que la bulle s'affiche puis se nettoie.
		script := fmt.Sprintf(`$ErrorActionPreference='SilentlyContinue'
Add-Type -AssemblyName System.Windows.Forms,System.Drawing
$n=New-Object System.Windows.Forms.NotifyIcon
$n.Icon=[System.Drawing.SystemIcons]::Information
$n.Visible=$true
$n.ShowBalloonTip(6000, %s, %s, 'Info')
Start-Sleep -Milliseconds 6000
$n.Dispose()`, psQuote(title), psQuote(body))
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)

	case "darwin":
		script := fmt.Sprintf(`display notification %s with title %s sound name "Glass"`,
			asQuote(body), asQuote(title))
		return exec.Command("osascript", "-e", script)

	default: // Linux
		return exec.Command("notify-send", title, body)
	}
}

// psQuote met une chaîne entre apostrophes PowerShell, où l'on échappe une
// apostrophe en la doublant. Suffisant pour glisser un titre quelconque sans
// risque d'injection dans le script.
func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// asQuote fait de même pour AppleScript, qui échappe avec un antislash.
func asQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
