# totp

Les codes à six chiffres qui changent toutes les trente secondes —
Google Authenticator, Authy, et compagnie — paraissent magiques. Ils ne le
sont pas : c'est un HMAC du temps, tronqué. `totp` les génère depuis un
secret, en CLI. De quoi récupérer un accès quand le téléphone est resté à
la maison, ou simplement comprendre ce qui se passe sous le capot.

## Le cheminement

### Deux RFC empilées, et c'est tout

TOTP (RFC 6238) n'invente presque rien : c'est **HOTP** (RFC 4226) dont le
compteur est le temps. HOTP, lui, tient en trois gestes :

1. **HMAC** du compteur (8 octets, gros-boutiste) avec le secret comme clé.
2. **Troncature dynamique** : les 4 bits de poids faible du dernier octet
   du condensé donnent un offset, on lit 31 bits à partir de là. Pourquoi
   un offset variable ? Pour qu'aucune partie fixe du HMAC ne soit
   systématiquement exposée.
3. **Modulo** 10^chiffres pour obtenir le code décimal.

TOTP ajoute une seule ligne au-dessus : `compteur = temps_unix / période`.
La beauté du truc, c'est que l'implémentation correcte est *courte* — le
risque n'est pas la complexité, c'est de se tromper d'un bit.

### Le bit de poids fort qu'on masque

Dans la troncature, on lit quatre octets mais on force à zéro le bit de
poids fort du premier (`& 0x7f`). La RFC le fait pour éviter les histoires
de signe selon les langages. En Go, `uint32` est déjà non signé, mais je
garde le masque : coller à la spec au bit près, c'est ce qui fait qu'un
code généré ici est accepté par un vrai serveur. Sur un OTP, « presque
juste » égale « refusé ».

### La vérification : les vecteurs de la RFC

Plutôt que de me fier à « ça compile et ça sort six chiffres », j'ai câblé
les **vecteurs de test officiels de la RFC 6238** (annexe B) : des couples
(instant, code attendu) sur les trois algorithmes, à des dates allant de
1970 à l'an 2603 (le fameux dépassement de `time_t` sur 32 bits). Si mon
code reproduit ces valeurs au chiffre près, c'est qu'il est conforme — et
qu'il interopérera avec Google, GitHub et les autres. C'est le test qui
m'a fait dormir tranquille.

### Le secret tel qu'on le copie réellement

Les sites affichent le secret en **base32**, souvent par groupes de quatre
(`JBSW Y3DP EHPK 3PXP`), parfois en minuscules, avec ou sans padding `=`.
Si je n'accepte que la forme canonique, l'outil est pénible à chaque
usage. Donc je nettoie avant de décoder : majuscules, espaces retirés,
padding optionnel. L'utilisateur colle ce qu'il a sous les yeux, ça marche.

### Lire le QR code, pas seulement le secret

Quand on scanne un QR d'authentification, ce n'est pas qu'un secret :
c'est une URI `otpauth://totp/Label?secret=…&digits=…&period=…&algorithm=…`
qui transporte *tous* les paramètres. `totp` détecte ce format et lit ces
réglages directement — inutile de deviner que tel site utilise 8 chiffres
ou des fenêtres de 60 secondes, l'URI le dit. Les valeurs absentes
retombent sur les défauts de la spec (6 chiffres, 30 s, SHA1), qui sont
aussi ceux de l'écrasante majorité des sites.

### Le mode `-watch`

Un code TOTP, c'est vivant : il expire. En mode `-watch`, je réimprime sur
la même ligne (`\r`) le code et une petite barre qui se vide à mesure que
la fenêtre se referme. On voit d'un coup d'œil s'il reste le temps de le
saisir ou s'il vaut mieux attendre le prochain.

## Ce que j'ai laissé tomber

- **HOTP en sortie directe** (le compteur d'événements de la RFC 4226).
  La brique est là, mais l'usage courant à la main, c'est le TOTP basé sur
  le temps. Je refuse d'ailleurs explicitement les URI `otpauth://hotp/`
  plutôt que de faire semblant.
- **Le stockage des secrets** (un coffre chiffré, une liste de comptes).
  Garder des secrets 2FA en clair sur disque, c'est précisément ce qu'il
  ne faut pas faire à la légère ; ça mérite son propre projet, pas un
  fichier de config improvisé.
- **La génération de QR / l'enregistrement de comptes.** C'est le rôle du
  service distant ; ici on consomme un secret existant. (Et pour le QR,
  il y a déjà `day05-qr`.)

## Usage

```
totp <secret base32> [options]
totp <otpauth://totp/...>          # les paramètres viennent de l'URI
```

Options (ignorées si on passe une URI otpauth) :

```
-digits <n>    nombre de chiffres (déf. 6)
-period <s>    durée de validité en secondes (déf. 30)
-algo <nom>    SHA1, SHA256 ou SHA512 (déf. SHA1)
-watch         rafraîchir le code en continu (Ctrl-C pour sortir)
```

Exemples :

```
totp JBSWY3DPEHPK3PXP
totp "JBSW Y3DP EHPK 3PXP" -watch
totp "otpauth://totp/ACME:alice@acme.com?secret=JBSWY3DPEHPK3PXP&digits=8&period=60"
```

## Organisation

```
main.go   CLI : secret ou URI, affichage ponctuel ou mode -watch
totp.go   HOTP/TOTP, décodage base32, lecture d'URI otpauth
```
