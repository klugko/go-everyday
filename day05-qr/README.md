# qr

Générer un QR code, c'est l'archétype du truc qu'on délègue : un site web,
une lib qu'on importe sans regarder, et hop. Sauf que la règle du repo,
c'est zéro dépendance. Alors soit je trichais, soit j'écrivais un encodeur
QR de zéro. J'ai écrit l'encodeur. `qr` affiche le code dans le terminal,
l'exporte en PNG, et sait fabriquer un QR de connexion Wi-Fi.

## Le cheminement

Un QR code, vu de loin, c'est un joli motif. Vu de près, c'est quatre
étapes qui s'enchaînent, et chacune a sa logique propre :

1. **Encoder** le texte en bits — un indicateur de mode, un compteur, les
   octets.
2. **Protéger** ces bits avec de la correction d'erreur Reed-Solomon, pour
   que le code reste lisible taché, plié ou à moitié caché.
3. **Poser** le tout dans la grille selon un parcours en zigzag bien
   précis, autour des motifs fixes (les trois « yeux », le timing,
   l'alignement).
4. **Masquer** : appliquer un des huit motifs de masquage qui rend le
   résultat le plus lisible, et l'inscrire dans l'info de format.

Rien d'insurmontable pris un par un, mais le diable est dans
l'enchaînement — une case de format mal réservée et tout le flux de
données se décale d'un cran (j'y reviens).

### Un seul mode d'encodage : octet

Le standard prévoit plusieurs modes — numérique, alphanumérique, octet,
kanji — chacun plus compact pour son type de contenu. J'ai tout misé sur
le **mode octet**. C'est le moins dense, mais c'est le seul qui encode
*n'importe quoi* : une URL, un mot de passe avec des symboles, du texte
accentué en UTF-8. Pour un outil généraliste, la simplicité d'un mode
unique qui ne se trompe jamais vaut mieux que trois modes à arbitrer pour
grappiller quelques octets.

### La correction d'erreur, là où ça devient mathématique

C'est le cœur du sujet, et la partie la moins intuitive. La correction
Reed-Solomon vit dans **GF(256)**, un corps fini où additionner revient à
faire un XOR et où multiplier est un produit de polynômes réduit. On y
calcule, pour chaque bloc de données, des codewords de redondance qui
permettront à un lecteur de reconstituer ce qui manque.

Deux détails comptent. D'abord la **multiplication** : écrite avec un
octet qui déborde « naturellement », elle tient en cinq lignes (voir le
commentaire dans `matrix.go`, c'est le genre d'astuce qui paraît magique
tant qu'on n'a pas posé le calcul). Ensuite l'**entrelacement** : on ne
colle pas les blocs bout à bout, on les mélange colonne par colonne. Comme
ça une tache concentrée sur le code n'abîme pas un bloc en entier mais un
peu de chacun — et chacun a de quoi se réparer.

### Le masquage, et pourquoi en essayer huit

Un QR rempli de grandes zones uniformes ou de motifs qui ressemblent aux
yeux de détection, c'est un cauchemar pour un scanner. D'où le masquage :
on inverse certains modules selon une formule géométrique pour casser ces
régularités. Le standard en définit huit, et la bonne pratique est de
**toutes les essayer** puis de garder celle qui obtient le plus petit
score de pénalité (séries trop longues, blocs uniformes, faux yeux,
déséquilibre noir/blanc). C'est un peu brutal — on construit huit grilles
pour n'en garder qu'une — mais à cette taille c'est instantané, et n'importe
lequel des huit masques donne un code valide : on optimise juste le confort
du lecteur.

### Le piège qui m'a coûté une heure

Tout passait mes tests « de structure », mais aucun lecteur ne décodait le
résultat. Le coupable : **une seule case**. L'info de format s'écrit en
deux exemplaires, et le second se découpe en 7 modules sur une colonne et
8 sur une ligne. J'avais inversé le découpage (8 + 7). Conséquence : une
case que je croyais libre était en fait réservée au format, donc tout mon
flux de bits de données se décalait d'un module à partir de là. La leçon :
sur un QR, « presque juste » égale « illisible ». J'ai fini par comparer ma
grille module par module à celle d'un encodeur de référence pour isoler la
case fautive.

### La vérification

Comme « ça compile et les yeux sont là » ne prouve rien, j'ai vérifié la
sortie en la faisant **décoder par un vrai lecteur** (zxing, celui qu'on
retrouve dans beaucoup d'applis téléphone) sur tout l'éventail : versions 1
à 10, les quatre niveaux de correction, du texte accentué, des
identifiants Wi-Fi avec caractères spéciaux. Le test golden du dépôt fige
une de ces grilles vérifiées pour attraper toute régression.

### Afficher dans un terminal

Un module noir, deux espaces de fond sombre ; un module blanc, deux
espaces de fond clair. Deux caractères de large parce qu'une cellule de
terminal est environ deux fois plus haute que large — le module ressort à
peu près carré. Je **force** les couleurs en ANSI plutôt que de me reposer
sur le thème : sur un terminal sombre, la marge claire obligatoire (la
« zone de silence ») disparaîtrait et le scanner ne verrait plus rien.
Cette marge fait 4 modules dans le PNG, mais seulement 2 au terminal —
sinon, deux espaces par module, la largeur explose et le code se replie.

### Le bonus Wi-Fi

Les téléphones savent lire une chaîne `WIFI:T:WPA;S:<réseau>;P:<mot de
passe>;;` et proposer la connexion sans rien saisir. Le seul vrai écueil,
c'est l'**échappement** : un SSID ou un mot de passe qui contient `;` `,`
`:` `"` ou `\` casserait la chaîne, donc on préfixe ces caractères d'un
antislash.

## Ce que j'ai laissé tomber

- **Les versions 11 à 40.** Je m'arrête à la version 10, qui encaisse déjà
  ~210 octets en niveau M — de quoi loger n'importe quelle URL ou clé
  Wi-Fi. Aller plus loin, c'est cinq fois plus de tables pour des cas qui
  n'arrivent jamais au terminal (et un QR illisible à l'écran de toute
  façon).
- **Les modes numérique, alphanumérique et kanji.** Plus compacts, mais le
  mode octet encode déjà tout. Le gain de place ne valait pas trois
  chemins de code à maintenir.
- **L'ECI et le « structured append ».** Encodages exotiques et codes
  chaînés sur plusieurs symboles : hors sujet pour un générateur du
  quotidien.
- **Les couleurs, logos et autres fioritures du PNG.** Noir sur blanc, net,
  point. C'est ce qui se scanne le mieux.

## Usage

```
qr [options] <texte>
qr -wifi -ssid <réseau> -pass <mot de passe> [options]
```

 Les options viennent **avant** le texte (`flag` s'arrête au premier
argument qui n'est pas une option).

Options :

```
-ec <niveau>    correction d'erreur : L, M, Q ou H (déf. M)
-png <chemin>   exporte aussi un PNG
-scale <n>      taille d'un module en pixels pour le PNG (déf. 8)
-wifi           mode Wi-Fi
-ssid <nom>     Wi-Fi : nom du réseau
-pass <mdp>     Wi-Fi : mot de passe
-enc <type>     Wi-Fi : WPA, WEP ou nopass (déf. WPA)
-hidden         Wi-Fi : réseau masqué
```

Exemples :

```
# Une URL dans le terminal
qr "https://example.com"

# Avec correction maximale et export PNG
qr -ec H -png url.png "https://example.com"

# Un QR de connexion Wi-Fi
qr -wifi -ssid "Maison_5G" -pass "monMotDePasse" -png wifi.png

# Un réseau ouvert (sans mot de passe)
qr -wifi -ssid "CaféGratuit" -enc nopass
```

## Organisation

```
main.go     CLI : flags, validation, aiguillage texte / Wi-Fi
qr.go       pipeline d'encodage : tables, bits, blocs + entrelacement
matrix.go   Galois/Reed-Solomon, pose dans la grille, masquage, pénalité
render.go   sortie terminal (ANSI), export PNG, chaîne Wi-Fi
```
