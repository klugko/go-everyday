package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Frankfurter expose les taux de référence de la BCE, gratuitement et sans
// clé d'API — exactement ce qu'il faut pour un petit outil : pas de secret à
// gérer, rien à configurer. La contrepartie : une trentaine de devises
// majeures seulement, et un taux quotidien (pas l'intraday).
const ratesAPI = "https://api.frankfurter.app/latest"

// rateResp est le morceau de la réponse JSON qui nous intéresse.
type rateResp struct {
	Base  string             `json:"base"`
	Date  string             `json:"date"`
	Rates map[string]float64 `json:"rates"`
}

// ConvertCurrency interroge le service de taux et convertit. Les codes sont
// mis en majuscules (norme ISO 4217) ; from==to court-circuite l'appel
// réseau, inutile de déranger l'API pour une identité.
func ConvertCurrency(value float64, from, to string) (Result, error) {
	from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
	if from == to {
		return Result{Value: value, FromSym: from, ToSym: to}, nil
	}

	resp, err := fetchRates(from, to)
	if err != nil {
		return Result{}, err
	}
	out, err := applyRate(value, to, resp)
	if err != nil {
		return Result{}, err
	}
	return Result{Value: out, FromSym: from, ToSym: to, Note: "taux BCE du " + resp.Date}, nil
}

func fetchRates(from, to string) (rateResp, error) {
	client := http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s?from=%s&to=%s", ratesAPI, from, to)

	r, err := client.Get(url)
	if err != nil {
		return rateResp{}, fmt.Errorf("appel au service de taux impossible : %w", err)
	}
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	// L'API renvoie 404 + un message quand la devise source est inconnue.
	if r.StatusCode != http.StatusOK {
		return rateResp{}, fmt.Errorf("devise %q refusée par le service (HTTP %d)", from, r.StatusCode)
	}
	return parseRates(body)
}

// parseRates est isolée de l'appel réseau pour rester testable avec une
// réponse figée.
func parseRates(body []byte) (rateResp, error) {
	var resp rateResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return rateResp{}, fmt.Errorf("réponse du service illisible : %w", err)
	}
	return resp, nil
}

func applyRate(value float64, to string, r rateResp) (float64, error) {
	rate, ok := r.Rates[to]
	if !ok {
		return 0, fmt.Errorf("devise %q non gérée par le service de taux", to)
	}
	return value * rate, nil
}
