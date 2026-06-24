# day25 — stego

Chiffrer un message le rend illisible, mais visible : « tiens, un fichier
chiffré ». La stéganographie joue l'inverse — le message est en clair, mais
personne ne soupçonne qu'il est là. `stego` glisse un texte dans un PNG sans
qu'on voie la moindre différence, puis le ressort.

```
stego hide photo.png sortie.png "rendez-vous à minuit"
stego hide photo.png sortie.png < lettre.txt     # message = fichier entier
stego reveal sortie.png                           # ressort le message sur stdout
```

## Décisions qui ont compté

- **LSB : le bit de poids faible de chaque canal.** Une composante rouge à 137
  ou 136, l'œil ne fait pas la différence. On planque donc un bit de message dans
  le LSB de chaque canal R, G, B. Trois bits cachés par pixel, pour une image
  qui semble identique — les tests vérifient qu'aucun canal ne bouge de plus de ±1.
- **PNG, parce que c'est sans perte.** En JPEG, la compression réécrit les pixels
  et efface aussitôt les bits qu'on vient de poser. Le PNG conserve chaque octet
  au ré-encodage : ce qu'on cache se relit à l'identique. C'est non négociable
  pour cette méthode.
- **Un en-tête de longueur, pas un marqueur de fin.** Les quatre premiers octets
  cachés disent combien d'octets suivent. À la lecture on sait exactement où
  s'arrêter, sans risquer de confondre un octet du message avec une « fin ». Une
  image vierge donne une longueur nulle → message vide, pas un plantage.
- **On saute le canal alpha.** Toucher la transparence d'une image à fond
  transparent se verrait (bords qui scintillent). On ne porte les bits que sur
  R, G, B ; l'alpha reste intact. La capacité est donc `largeur × hauteur × 3` bits.
- **On ne mute jamais l'image source.** `hide` part d'une copie NRGBA fraîche.
  NRGBA (non prémultipliée) permet de modifier R, G, B sans recalculer quoi que
  ce soit en fonction de l'alpha — le format manipulé bit à bit le plus simple.
- **La capacité borne la lecture.** Une image sans message donne un en-tête
  farfelu ; on refuse toute longueur dépassant ce que l'image peut contenir,
  plutôt que d'allouer des mégaoctets pour lire du bruit.

## Ce que j'ai laissé tomber

- **Le chiffrement du message.** Caché ne veut pas dire chiffré : qui connaît la
  méthode relit tout. Pour un vrai secret, on chiffre d'abord (voir
  [day24-enc](../day24-enc)) puis on cache le résultat. Combiner les deux outils
  plutôt que mélanger les rôles.
- **La résistance à l'analyse statistique.** Le LSB simple se détecte avec des
  outils dédiés (les histogrammes de LSB trahissent un remplissage régulier).
  Disperser les bits avec une clé pseudo-aléatoire serait plus discret — un cran
  au-dessus, hors périmètre du jour.
- **Les autres formats (BMP, GIF).** Tout format sans perte conviendrait, mais le
  PNG couvre le besoin et la stdlib le décode déjà. Un format de plus, pas une
  idée de plus.
- **Le multi-bits par canal.** Cacher 2 ou 3 bits par canal triple la capacité
  mais commence à se voir. Un bit, c'est le compromis invisible/capacité que je
  voulais.
