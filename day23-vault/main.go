// vault — coffre-fort de secrets chiffré (AES-GCM), déverrouillé par un mot de
// passe maître. Un seul fichier sur disque, illisible sans le mot de passe.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const defaultFile = "vault.dat"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "vault :", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "vault — coffre-fort de secrets chiffré par un mot de passe maître.")
	fmt.Fprintln(os.Stderr, "usage : vault <commande> [args]")
	fmt.Fprintln(os.Stderr, "  set <nom> [valeur]   range un secret (valeur lue sur stdin si absente)")
	fmt.Fprintln(os.Stderr, "  get <nom>            affiche un secret")
	fmt.Fprintln(os.Stderr, "  list                 liste les noms (jamais les valeurs)")
	fmt.Fprintln(os.Stderr, "  rm <nom>             supprime un secret")
	fmt.Fprintln(os.Stderr, "le coffre est ./"+defaultFile+" ; le mot de passe vient de $VAULT_PASSWORD ou est demandé.")
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("commande manquante")
	}
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "set":
		return cmdSet(rest)
	case "get":
		return cmdGet(rest)
	case "list", "ls":
		return cmdList()
	case "rm", "del":
		return cmdRm(rest)
	default:
		usage()
		return fmt.Errorf("commande inconnue %q", cmd)
	}
}

// load ouvre le coffre s'il existe, sinon part d'un coffre vide : le premier
// `set` créera le fichier. C'est ce qui rend l'initialisation transparente.
func load(password string) (map[string]string, error) {
	data, err := os.ReadFile(defaultFile)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return open(data, password)
}

// save réécrit tout le coffre, chiffré. On passe par un fichier temporaire puis
// un rename atomique : une coupure en plein milieu ne laisse pas un coffre à
// moitié écrit, donc illisible.
func save(secrets map[string]string, password string) error {
	blob, err := seal(secrets, password)
	if err != nil {
		return err
	}
	tmp := defaultFile + ".tmp"
	if err := os.WriteFile(tmp, blob, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, defaultFile)
}

func cmdSet(args []string) error {
	if len(args) < 1 {
		return errors.New("usage : vault set <nom> [valeur]")
	}
	name := args[0]

	var value string
	if len(args) >= 2 {
		value = strings.Join(args[1:], " ")
	} else {
		// Pas de valeur en argument : on la lit sur stdin pour qu'elle ne traîne
		// pas dans l'historique du shell.
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		value = strings.TrimRight(string(raw), "\r\n")
	}

	pw, err := masterPassword()
	if err != nil {
		return err
	}
	secrets, err := load(pw)
	if err != nil {
		return err
	}
	secrets[name] = value
	if err := save(secrets, pw); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "secret %q enregistré\n", name)
	return nil
}

func cmdGet(args []string) error {
	if len(args) != 1 {
		return errors.New("usage : vault get <nom>")
	}
	pw, err := masterPassword()
	if err != nil {
		return err
	}
	secrets, err := load(pw)
	if err != nil {
		return err
	}
	value, ok := secrets[args[0]]
	if !ok {
		return fmt.Errorf("aucun secret nommé %q", args[0])
	}
	// La valeur seule sur stdout : on peut faire `vault get X | pbcopy`.
	fmt.Println(value)
	return nil
}

func cmdList() error {
	pw, err := masterPassword()
	if err != nil {
		return err
	}
	secrets, err := load(pw)
	if err != nil {
		return err
	}
	if len(secrets) == 0 {
		fmt.Fprintln(os.Stderr, "coffre vide")
		return nil
	}
	names := make([]string, 0, len(secrets))
	for n := range secrets {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

func cmdRm(args []string) error {
	if len(args) != 1 {
		return errors.New("usage : vault rm <nom>")
	}
	pw, err := masterPassword()
	if err != nil {
		return err
	}
	secrets, err := load(pw)
	if err != nil {
		return err
	}
	if _, ok := secrets[args[0]]; !ok {
		return fmt.Errorf("aucun secret nommé %q", args[0])
	}
	delete(secrets, args[0])
	if err := save(secrets, pw); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "secret %q supprimé\n", args[0])
	return nil
}

// masterPassword récupère le mot de passe maître. On privilégie la variable
// d'environnement (scriptable, testable) ; à défaut on demande sur le terminal.
// L'écho n'est pas coupé — la stdlib ne le fait pas en portable — donc pour un
// usage discret, mieux vaut passer par $VAULT_PASSWORD.
func masterPassword() (string, error) {
	if pw := os.Getenv("VAULT_PASSWORD"); pw != "" {
		return pw, nil
	}
	fmt.Fprint(os.Stderr, "mot de passe maître : ")
	var pw string
	if _, err := fmt.Fscanln(os.Stdin, &pw); err != nil {
		return "", errors.New("mot de passe vide")
	}
	if pw == "" {
		return "", errors.New("mot de passe vide")
	}
	return pw, nil
}
