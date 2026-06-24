# day24 — enc

Envoyer un fichier sensible par mail ou le déposer sur un drive, c'est le donner
en clair à tout le monde sur le chemin. `enc` le chiffre avec un mot de passe
avant le départ ; le destinataire le déchiffre avec le même mot de passe. Rien à
installer des deux côtés à part le binaire.

```
enc secret.pdf                 # → secret.pdf.enc
enc -d secret.pdf.enc          # → secret.pdf
enc -d -o clair.pdf x.enc      # nom de sortie forcé
```

Le mot de passe vient de `-p`, sinon de `$ENC_PASSWORD`, sinon il est demandé.

## Décisions qui ont compté

- **AES-256-GCM, le même socle que [le coffre du jour 23](../day23-vault).** GCM
  chiffre *et* authentifie : déchiffrer avec un mauvais mot de passe, ou sur un
  fichier modifié d'un seul octet, échoue net. On ne livre jamais un déchiffré
  silencieusement faux.
- **La clé dérive du mot de passe via PBKDF2.** Un mot de passe n'est pas une clé.
  PBKDF2 (200 000 itérations, SHA-256, depuis `crypto/pbkdf2` de la stdlib)
  l'étire en clé AES et rend le brute-force coûteux. Sel neuf à chaque fichier,
  rangé en tête : deux fichiers identiques chiffrés donnent deux sorties distinctes.
- **Un petit magic en tête.** Les quatre octets `ENC1` permettent de reconnaître
  nos fichiers et de répondre « ce fichier n'a pas été chiffré par enc » au lieu
  d'un cryptique échec de déchiffrement quand on pointe `-d` sur un PDF normal.
- **On refuse d'écraser.** Si la sortie existe déjà, on s'arrête. Oublier le `-d`
  ne doit jamais détruire l'original en le « chiffrant » par-dessus. Sortie en
  `0600`, comme le coffre.
- **Tout en mémoire, fichier entier.** On lit, on (dé)chiffre, on écrit. Simple,
  lisible, suffisant pour les fichiers qu'on partage à la main. Le streaming par
  blocs viendrait pour des fichiers de plusieurs Go — pas le cas d'usage ici.

## Ce que j'ai laissé tomber

- **Le chiffrement en flux (streaming).** GCM en un bloc impose de tout charger
  en mémoire. Pour de très gros fichiers il faudrait découper en segments
  numérotés, chacun avec son tag, pour résister au réordonnancement. C'est un
  vrai sujet, mais un autre outil.
- **La compression avant chiffrement.** Tentant pour la taille, mais ça ouvre la
  porte aux attaques par oracle de compression. Hors périmètre, et pas sans risque.
- **Les clés publiques (partage asymétrique).** Ici c'est un secret partagé : les
  deux bouts connaissent le mot de passe. Chiffrer pour la clé publique de
  quelqu'un, c'est le terrain de GPG/age — pas ce jour.
- **L'écho coupé à la saisie.** Comme au jour 23 : la stdlib ne le fait pas en
  portable. `$ENC_PASSWORD` couvre les scripts ; le reste resterait à la saisie.
