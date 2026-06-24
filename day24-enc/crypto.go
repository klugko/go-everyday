package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

// Format d'un fichier chiffré : magic | sel | nonce | chiffré+tag.
// Le magic sert juste à reconnaître nos fichiers et à refuser de déchiffrer
// n'importe quoi avec un message d'erreur clair.

var magic = []byte("ENC1")

const (
	saltLen = 16
	iter    = 200000 // PBKDF2 : étire le mot de passe et ralentit le brute-force
	keyLen  = 32     // AES-256
)

var (
	errBadPassword  = errors.New("mot de passe incorrect ou fichier corrompu")
	errNotEncrypted = errors.New("ce fichier n'a pas été chiffré par enc")
)

func deriveKey(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, iter, keyLen)
}

// encrypt chiffre des octets quelconques. On tire un sel et un nonce neufs à
// chaque appel : le même fichier chiffré deux fois donne deux sorties distinctes.
func encrypt(plain []byte, password string) ([]byte, error) {
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
	sealed := gcm.Seal(nil, nonce, plain, nil)

	out := make([]byte, 0, len(magic)+saltLen+len(nonce)+len(sealed))
	out = append(out, magic...)
	out = append(out, salt...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

// decrypt refait le chemin inverse et vérifie l'intégrité via le tag GCM.
func decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < len(magic) || string(data[:len(magic)]) != string(magic) {
		return nil, errNotEncrypted
	}
	data = data[len(magic):]
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
	nonce, sealed := rest[:ns], rest[ns:]

	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, errBadPassword
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
