package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	args := os.Args[1:]
	if len(args) != 3 {
		usage()
		os.Exit(2)
	}

	value, err := strconv.ParseFloat(strings.Replace(args[0], ",", ".", 1), 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conv: valeur invalide %q\n", args[0])
		os.Exit(1)
	}
	from, to := args[1], args[2]

	// On essaie d'abord hors-ligne (longueurs, poids, températures). Si les
	// unités n'en relèvent pas, on suppose des devises et on passe au réseau.
	res, ok, err := ConvertOffline(value, from, to)
	if !ok {
		res, err = ConvertCurrency(value, from, to)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "conv:", err)
		os.Exit(1)
	}

	line := fmt.Sprintf("%s %s = %s %s", num(value), res.FromSym, num(res.Value), res.ToSym)
	if res.Note != "" {
		line += "  (" + res.Note + ")"
	}
	fmt.Println(line)
}

// num formate sans bruit : on coupe à six décimales puis on retire les zéros
// inutiles, pour que 212.000000 s'affiche 212 et 6.213712 reste intact.
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func usage() {
	fmt.Fprint(os.Stderr, `conv — convertisseur universel

  conv <valeur> <de> <vers>

Longueurs : mm cm dm m km in ft yd mi nmi
Poids     : mg g kg t oz lb st
Température: c f k  (°C, fahrenheit, kelvin… acceptés)
Devises   : codes ISO sur trois lettres, taux BCE en direct (USD, EUR, JPY, GBP…)

Exemples :
  conv 10 km mi
  conv 80 kg lb
  conv 100 c f
  conv 50 usd eur
`)
}
