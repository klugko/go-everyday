# day12 — fmtj

Convertir un fichier de config entre **JSON** et **YAML**, dans les deux sens,
sans ouvrir un éditeur en ligne louche pour ça.

## Le problème

On copie sans arrêt des bouts de config d'un format à l'autre : un exemple
trouvé en YAML qu'il faut coller dans un `package.json`, un retour d'API JSON
qu'on veut relire en YAML parce que c'est moins pénible. À la main c'est
fastidieux et on se trompe d'indentation.

```
fmtj config.json          # → YAML sur stdout (bascule par défaut)
fmtj config.yaml          # → JSON
fmtj --to json data.yml   # force la cible
cat data.json | fmtj --from json --to yaml
```

Sans fichier, lit stdin. Écrit toujours sur stdout — libre à toi de rediriger.

## Décisions qui ont compté

- **On passe par un `any` intermédiaire.** Décoder vers `any` puis réencoder,
  c'est tout. Le seul piège classique (yaml.v2 décode les mappings en
  `map[interface{}]interface{}` que `json.Marshal` refuse) n'existe pas avec
  **yaml.v3**, qui rend des `map[string]any`. D'où le choix de v3.
- **Bascule par défaut.** Sans `--to`, on convertit vers *l'autre* format —
  c'est 90 % des usages. `--from` se devine via l'extension du fichier.
- **JSON indenté à 2 espaces** en sortie, lisible et stable pour le round-trip.

## Ce que j'ai laissé tomber

- **TOML.** L'objectif initial parlait de JSON/YAML/TOML, mais TOML aurait
  ajouté une 2ᵉ dépendance et des cas tordus (tables, dates typées). Gardé
  pour plus tard si le besoin se présente.
- **Écriture en place** (`-i`). Une redirection shell fait le travail.

## Entorse à la règle « stdlib only »

Tous les jours précédents tiennent en stdlib pure. Ici c'est impossible :
YAML n'est pas dans la stdlib Go. Une seule dépendance assumée,
[`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) — la référence du
domaine.
