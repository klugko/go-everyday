# day19 — linkcheck

Trouver les liens morts dans un Markdown, ou dans tout un dossier de docs d'un
coup. Le genre de truc qu'on découvre trop tard : un fichier renommé, une URL
qui a bougé, et le README pointe dans le vide depuis des mois.

## Le problème

Une doc vit. On déplace `install.md`, on renomme une section, un site externe
ferme — et chaque lien qui pointait là devient mort sans prévenir. Personne ne
reclique tout à la main. Je voulais une commande à lancer en CI : elle sort en
erreur dès qu'un lien retombe dans le vide, et me dit lequel et où.

```
linkcheck README.md          # un seul fichier
linkcheck docs/              # tout le dossier, en récursif
linkcheck                    # le dossier courant par défaut
linkcheck -timeout 3s docs/  # patience plus courte sur les liens externes
```

La sortie pointe direct : `docs/intro.md:42  ./api.md  → fichier introuvable`.
Code de retour 1 s'il reste un mort, 0 si tout tient — de quoi casser un build.

## Décisions qui ont compté

- **Trois familles de liens, trois traitements.** Un lien local (`./api.md`,
  `img/logo.png`) se cherche sur le disque, relatif au fichier qui le cite. Un
  lien `http(s)` part sur le réseau. Le reste — ancres `#section`, `mailto:`,
  `tel:` — on n'a pas les moyens de le juger, donc on le laisse vivre plutôt que
  de crier au loup.
- **Le disque avant le réseau.** Le plus gros des liens morts, ce sont des
  fichiers déplacés, pas des sites tombés. Un `os.Stat` règle ça sans rien
  envoyer. On retire l'ancre et la query au passage (`api.md#tag` → `api.md`) :
  c'est le fichier qui compte, pas le fragment.
- **HEAD d'abord, GET en repli.** Pour l'externe, un HEAD suffit et ne tire pas
  la page entière. Sauf que pas mal de serveurs répondent 403 ou 405 à un HEAD
  qu'ils n'aiment pas — alors on réessaie en GET avant de conclure « mort ».
  Ça évite une pluie de faux positifs.
- **Un pool de goroutines.** Le réseau est lent et c'est lui qui dicte le tempo,
  donc on vérifie plusieurs liens en même temps (`-n`, 8 par défaut). Les
  résultats sont re-triés par fichier puis par ligne à la fin : l'ordre de
  sortie ne dépend pas de qui a répondu en premier.
- **On saute les blocs de code.** Un `[exemple](url)` dans un ` ``` ` est une
  démo, pas un lien à suivre. Même bascule sur les clôtures que les autres jours,
  pas de vrai parseur Markdown.

## Ce que j'ai laissé tomber

- **Les liens de référence** (`[texte][ref]` avec `[ref]: url` plus bas) et les
  autolinks `<https://…>`. Le `[texte](url)` inline couvre l'écrasante majorité
  de ce que j'écris ; les deux autres formes attendront un besoin réel.
- **Vérifier les ancres** (`#section` existe-t-elle vraiment dans la cible ?).
  C'est un autre métier — proche de [day18-toc](../day18-toc) qui sait déjà
  lister les titres. Ici on se contente du fichier.
- **Un cache des URL déjà vues.** Le même lien externe répété dans dix fichiers
  est tapé dix fois. À l'échelle d'un dossier de docs ça reste rapide ; un
  `map[url]résultat` serait l'optimisation suivante si ça coince.
- **Suivre les redirections jusqu'au bout / réessayer.** Le client Go suit déjà
  les 3xx normales. Au-delà (retry sur timeout, backoff), c'est du confort que
  je n'ai pas eu à payer.
