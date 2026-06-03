package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
)

// Algo désigne la fonction de hachage du HMAC. SHA1 est ce que produisent
// Google Authenticator et la quasi-totalité des sites.
type Algo int

const (
	SHA1 Algo = iota
	SHA256
	SHA512
)

func (a Algo) hash() func() hash.Hash {
	switch a {
	case SHA256:
		return sha256.New
	case SHA512:
		return sha512.New
	default:
		return sha1.New
	}
}

func (a Algo) String() string {
	switch a {
	case SHA256:
		return "SHA256"
	case SHA512:
		return "SHA512"
	default:
		return "SHA1"
	}
}

// HOTP calcule un code à `digits` chiffres pour un compteur donné
// (RFC 4226). C'est la brique sur laquelle repose TOTP.
func HOTP(secret []byte, counter uint64, digits int, algo Algo) string {
	mac := hmac.New(algo.hash(), secret)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	// Troncature dynamique : les 4 bits de poids faible du dernier octet
	// donnent un offset, on y lit 31 bits (le bit de poids fort masqué
	// pour rester positif quel que soit le langage).
	off := sum[len(sum)-1] & 0x0f
	code := uint32(sum[off]&0x7f)<<24 |
		uint32(sum[off+1])<<16 |
		uint32(sum[off+2])<<8 |
		uint32(sum[off+3])

	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, code%mod)
}

// TOTP, c'est HOTP indexé par le temps (RFC 6238) : le compteur est le
// nombre de tranches de `period` secondes écoulées depuis l'epoch Unix.
func TOTP(secret []byte, unix int64, period, digits int, algo Algo) string {
	return HOTP(secret, uint64(unix/int64(period)), digits, algo)
}

// DecodeSecret lit un secret base32 tel qu'on le copie depuis un site :
// minuscules tolérées, espaces de regroupement ignorés, padding optionnel.
func DecodeSecret(s string) ([]byte, error) {
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	s = strings.TrimRight(s, "=")
	if s == "" {
		return nil, fmt.Errorf("secret vide")
	}
	b, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("secret base32 invalide : %w", err)
	}
	return b, nil
}

// Config regroupe tout ce qu'il faut pour générer un code.
type Config struct {
	Secret []byte
	Digits int
	Period int
	Algo   Algo
	Label  string 
}

// ParseURI lit une URI otpauth://totp/... (le QR code que scanne une appli
// d'authentification). Les paramètres absents prennent les valeurs par
// défaut de la spec : 6 chiffres, 30 s, SHA1.
func ParseURI(raw string) (Config, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Config{}, fmt.Errorf("URI illisible : %w", err)
	}
	if u.Scheme != "otpauth" {
		return Config{}, fmt.Errorf("schéma %q inattendu (attendu otpauth)", u.Scheme)
	}
	if u.Host != "totp" {
		return Config{}, fmt.Errorf("type %q non géré (seul totp l'est)", u.Host)
	}

	q := u.Query()
	secret, err := DecodeSecret(q.Get("secret"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Secret: secret,
		Digits: 6,
		Period: 30,
		Algo:   SHA1,
		Label:  strings.TrimPrefix(u.Path, "/"),
	}
	if v := q.Get("digits"); v != "" {
		if cfg.Digits, err = strconv.Atoi(v); err != nil {
			return Config{}, fmt.Errorf("digits invalide : %w", err)
		}
	}
	if v := q.Get("period"); v != "" {
		if cfg.Period, err = strconv.Atoi(v); err != nil {
			return Config{}, fmt.Errorf("period invalide : %w", err)
		}
	}
	if v := q.Get("algorithm"); v != "" {
		if cfg.Algo, err = ParseAlgo(v); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

// ParseAlgo accepte les noms tels qu'ils apparaissent dans une URI.
func ParseAlgo(s string) (Algo, error) {
	switch strings.ToUpper(s) {
	case "SHA1", "":
		return SHA1, nil
	case "SHA256":
		return SHA256, nil
	case "SHA512":
		return SHA512, nil
	}
	return 0, fmt.Errorf("algorithme %q inconnu (SHA1, SHA256 ou SHA512)", s)
}
