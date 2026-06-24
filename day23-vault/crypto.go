package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
)

// Le coffre, une fois ouvert, n'est qu'une map nom → secret. Tout le reste
// (sel, nonce, chiffré) ne sert qu'à la stocker sur disque sans la révéler.

const (
	saltLen = 16     // sel unique par coffre, mélangé au mot de passe
	iter    = 200000 // itérations PBKDF2 : assez pour rendre le brute-force pénible
	keyLen  = 32     // 32 octets → AES-256
)

// errBadPassword remonte quand GCM refuse de déchiffrer : soit le mot de passe
// est faux, soit le fichier a été touché. On ne sait pas dire lequel, et c'est
// très bien ainsi — un attaquant n'apprend rien.
var errBadPassword = errors.New("mot de passe incorrect ou coffre corrompu")

// deriveKey transforme le mot de passe maître en clé AES. Le sel garantit que
// deux coffres au même mot de passe n'aboutissent pas à la même clé.
func deriveKey(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iter, keyLen)
}

// seal chiffre le coffre entier. Le fichier produit est : sel | nonce | chiffré.
// Sel et nonce sont en clair, c'est leur rôle ; seul le chiffré tient le secret.
func seal(secrets map[string]string, password string) ([]byte, error) {
	plain, err := json.Marshal(secrets)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// Seal colle le chiffré + le tag d'authentification après le nonce.
	blob := gcm.Seal(nil, nonce, plain, nil)

	out := make([]byte, 0, saltLen+len(nonce)+len(blob))
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, blob...)
	return out, nil
}

// open fait le chemin inverse : on relit le sel et le nonce, on redérive la clé
// depuis le mot de passe, puis GCM déchiffre *et* vérifie l'intégrité.
func open(data []byte, password string) (map[string]string, error) {
	if len(data) < saltLen {
		return nil, errBadPassword
	}
	salt, rest := data[:saltLen], data[saltLen:]

	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	ns := gcm.NonceSize()
	if len(rest) < ns {
		return nil, errBadPassword
	}
	nonce, blob := rest[:ns], rest[ns:]

	plain, err := gcm.Open(nil, nonce, blob, nil)
	if err != nil {
		// Mauvais mot de passe ou octet altéré : GCM tombe ici dans les deux cas.
		return nil, errBadPassword
	}

	secrets := map[string]string{}
	if err := json.Unmarshal(plain, &secrets); err != nil {
		return nil, fmt.Errorf("contenu illisible : %w", err)
	}
	return secrets, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
