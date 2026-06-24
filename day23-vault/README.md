# day23 — vault

Les secrets finissent toujours quelque part : un `.env` en clair, un Post-it, un
gestionnaire de mots de passe pour trois tokens d'API. `vault` est le minimum
honnête entre les deux — un seul fichier chiffré, déverrouillé par un mot de
passe maître, manipulé en ligne de commande.

```
export VAULT_PASSWORD=mon-mot-de-passe
vault set github ghp_xxx          # range un secret
vault set db                      # valeur lue sur stdin (pas dans l'historique)
vault get github                  # affiche la valeur, seule sur stdout
vault list                        # les noms, jamais les valeurs
vault rm db                       # supprime
```

Le coffre vit dans `./vault.dat`. Sans `$VAULT_PASSWORD`, le mot de passe est
demandé sur le terminal.

## Décisions qui ont compté

- **AES-256-GCM, pas juste AES.** GCM chiffre *et* authentifie : un octet modifié
  dans le fichier, et l'ouverture échoue franchement au lieu de rendre une bouillie.
  Le même tag qui détecte la corruption détecte le mauvais mot de passe — on ne
  sait pas distinguer les deux, et c'est voulu : un attaquant n'apprend rien.
- **La clé vient du mot de passe via PBKDF2, pas directement.** Un mot de passe
  n'est pas une clé de 32 octets. PBKDF2 (200 000 itérations, SHA-256) l'étire en
  une vraie clé AES et rend le brute-force coûteux. Tout est dans la stdlib depuis
  `crypto/pbkdf2` — aucune dépendance externe, comme les autres jours.
- **Un sel par coffre, un nonce par écriture.** Stockés en clair en tête de
  fichier, c'est leur rôle. Le sel fait que deux coffres au même mot de passe
  n'ont pas la même clé ; le nonce fait que deux écritures du même contenu donnent
  deux fichiers différents. Personne ne peut deviner « ces deux coffres se
  ressemblent ».
- **Écriture atomique.** On scelle dans `vault.dat.tmp` puis on `rename`. Une
  coupure de courant en plein `set` ne laisse jamais un coffre à moitié écrit,
  donc indéchiffrable. Le fichier est en `0600` : lisible par son seul propriétaire.
- **Le mot de passe passe par l'environnement.** `$VAULT_PASSWORD` rend l'outil
  scriptable et testable. Le repli interactif lit une ligne *avec* écho — la
  stdlib ne coupe pas l'écho de façon portable, et je n'allais pas tirer une
  dépendance juste pour ça. Pour un usage discret, on reste sur la variable.

## Ce que j'ai laissé tomber

- **Le multi-coffre et les chemins configurables.** Un fichier, au même endroit.
  Plusieurs coffres, c'est plusieurs dossiers — pas un drapeau de plus.
- **La rotation du mot de passe maître.** Re-chiffrer tout sous un nouveau mot de
  passe serait une commande de plus ; aujourd'hui on relit, on supprime, on
  recrée si besoin. Hors périmètre du jour.
- **L'écho coupé à la saisie.** Vrai manque pour un usage interactif, mais il
  demande du code spécifique par OS ou une dépendance. La variable d'environnement
  couvre le cas réel (CI, scripts) sans ça.
- **Le partage entre machines.** Le fichier est portable tel quel — on le copie,
  on connaît le mot de passe, on l'ouvre. La synchro, c'est le boulot d'un autre
  outil (git, rsync), pas du coffre.
