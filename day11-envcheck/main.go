package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	envPath := flag.String("env", ".env", "fichier à vérifier")
	examplePath := flag.String("example", ".env.example", "fichier modèle de référence")
	flag.Parse()

	example, err := os.ReadFile(*examplePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(2)
	}
	env, err := os.ReadFile(*envPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(2)
	}

	diff := Compare(Keys(string(example)), Keys(string(env)))

	if diff.OK() {
		fmt.Printf("%s est en phase avec %s.\n", *envPath, *examplePath)
		return
	}

	if len(diff.Missing) > 0 {
		fmt.Printf("Manquantes (dans %s, absentes de %s) :\n", *examplePath, *envPath)
		for _, k := range diff.Missing {
			fmt.Println("  -", k)
		}
	}
	if len(diff.Extra) > 0 {
		if len(diff.Missing) > 0 {
			fmt.Println()
		}
		fmt.Printf("En trop (dans %s, absentes de %s) :\n", *envPath, *examplePath)
		for _, k := range diff.Extra {
			fmt.Println("  +", k)
		}
	}

	// Code de sortie non nul : utilisable tel quel dans un CI.
	os.Exit(1)
}
