// fmtj — convertit entre JSON et YAML dans les deux sens.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	from := flag.String("from", "", "format d'entrée : json ou yaml (déduit de l'extension du fichier sinon)")
	to := flag.String("to", "", "format de sortie : json ou yaml (l'inverse de --from par défaut)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "fmtj — convertit entre JSON et YAML.")
		fmt.Fprintln(os.Stderr, "usage : fmtj [--from json|yaml] [--to json|yaml] [fichier]")
		fmt.Fprintln(os.Stderr, "sans fichier, lit l'entrée standard ; écrit toujours sur la sortie standard.")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := run(*from, *to, flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, "fmtj :", err)
		os.Exit(1)
	}
}

func run(from, to, file string) error {
	var (
		data []byte
		err  error
	)
	if file == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return err
	}

	// --from explicite l'emporte ; sinon on devine via l'extension.
	if from == "" {
		from = detectFormat(file)
	}
	if from == "" {
		return fmt.Errorf("format d'entrée indéterminé : précise --from json|yaml")
	}
	// Par défaut on bascule vers l'autre format
	if to == "" {
		to = opposite(from)
	}

	out, err := convert(data, from, to)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// convert décode data depuis from puis le réencode vers to. 
func convert(data []byte, from, to string) ([]byte, error) {
	var v any
	switch from {
	case "json":
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("JSON invalide : %w", err)
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("YAML invalide : %w", err)
		}
	default:
		return nil, fmt.Errorf("format d'entrée inconnu : %q (json ou yaml)", from)
	}

	switch to {
	case "json":
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(out, '\n'), nil
	case "yaml":
		return yaml.Marshal(v)
	default:
		return nil, fmt.Errorf("format de sortie inconnu : %q (json ou yaml)", to)
	}
}

// detectFormat devine le format depuis l'extension du fichier, "" si inconnu.
func detectFormat(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	}
	return ""
}

func opposite(f string) string {
	if f == "json" {
		return "yaml"
	}
	return "json"
}
