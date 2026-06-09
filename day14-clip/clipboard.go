package main

import (
	"os/exec"
	"runtime"
	"strings"
)


// readClipboard renvoie le contenu courant du presse-papiers (texte).
func readClipboard() (string, error) {
	out, err := readCmd().Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// writeClipboard remplace le contenu du presse-papiers par text.
func writeClipboard(text string) error {
	cmd := writeCmd()
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func readCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		// -Raw garde le texte tel quel, retours à la ligne compris.
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-Clipboard -Raw")
	case "darwin":
		return exec.Command("pbpaste")
	default: 
	// Linux : xclip est le plus répandu pour la sélection "clipboard".
		return exec.Command("xclip", "-selection", "clipboard", "-o")
	}
}

func writeCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "$input | Set-Clipboard")
	case "darwin":
		return exec.Command("pbcopy")
	default:
		return exec.Command("xclip", "-selection", "clipboard")
	}
}
