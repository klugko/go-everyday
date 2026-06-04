# conv

Un convertisseur, ça paraît trivial — multiplier par un facteur, fin de
l'histoire. Sauf que « universel » cache trois pièges : les températures ne
sont pas linéaires, les devises bougent toutes les heures, et l'utilisateur
ne veut surtout pas préciser de quel *type* d'unité il parle. `conv` prend
`<valeur> <de> <vers>` et se débrouille pour deviner le reste.

## Le cheminement

### Une seule table, devinée par les unités elles-mêmes

Premier réflexe : un sous-commande par domaine (`conv length …`,
`conv temp …`). Pénible à taper, et ça oblige l'utilisateur à savoir d'avance
que « lb » est un poids. À la place, les unités *portent* leur dimension. Je
mets longueurs et poids dans une même table `map[string]unit`, chaque entrée
connaissant son facteur vers une unité de base (le mètre, le gramme) et son
domaine. Convertir, c'est alors une ligne :

```
résultat = valeur * facteur_source / facteur_cible
```

Le passage par une base commune évite d'écrire toutes les paires possibles —
ajouter le mille marin, c'est une ligne, pas dix.

### La dimension comme garde-fou

Comme chaque unité connaît son domaine, `km → kg` n'est pas un calcul faux
qui passe inaperçu : c'est une **erreur explicite**. C'est tout l'intérêt de
ranger la dimension dans la table plutôt que de jongler avec des facteurs nus.

### La température refuse d'être linéaire

Celsius, Fahrenheit et Kelvin n'ont pas la même origine : on ne peut pas s'en
sortir avec un simple facteur. Je les sors donc de la table et je pivote
systématiquement **par le Celsius** : `source → °C → cible`. Deux petites
fonctions au lieu d'un tableau de toutes les paires. Le test qui me rassure,
c'est `-40 °C = -40 °F` — le point où les deux échelles se croisent : si je me
suis trompé d'un signe, il saute.

### Les devises : en direct, mais sans clé

« En direct » sous-entend une API. J'ai choisi **Frankfurter** (les taux de
référence de la BCE) pour une raison simple : *aucune clé*. Pas de secret à
stocker, rien à configurer, l'outil marche dès le `go build`. La contrepartie
assumée : une trentaine de devises majeures et un taux quotidien, pas
l'intraday. Pour un convertisseur de poche, c'est exactement le bon compromis.

### Hors-ligne d'abord, réseau en dernier recours

Le dispatch tient en trois lignes : on tente la conversion hors-ligne
(longueurs, poids, températures) ; si *aucune* des deux unités n'en relève, on
suppose des devises et on appelle le réseau. Conséquence agréable : tout ce
qui est physique reste instantané et testable sans connexion, et on ne sollicite
l'API que quand c'est vraiment des codes monétaires. Le décodage du JSON et
l'application du taux sont isolés de l'appel HTTP, histoire de les tester avec
une réponse figée.

## Ce que j'ai laissé tomber

- **Les sous-commandes et les flags de domaine.** La détection par l'unité
  rend tout ça superflu, et `conv 10 km mi` se tape sans réfléchir.
- **Un cache des taux sur disque.** Tentant pour le mode avion, mais ça
  introduit la question « jusqu'à quand un taux est-il encore valable ? ».
  Hors sujet pour un outil qui dit « en direct » dans son cahier des charges.
- **Les conversions exotiques** (volumes, surfaces, vitesses, octets…). La
  mécanique est là : ajouter une dimension, c'est étendre la table. J'ai
  préféré quatre domaines solides qu'une liste à rallonge à moitié testée.
- **Les fractions impériales** (`5'9"`). Joli, mais c'est un parseur d'entrée
  à part entière pour un cas d'usage de niche.

## Usage

```
conv <valeur> <de> <vers>
```

```
Longueurs : mm cm dm m km in ft yd mi nmi
Poids     : mg g kg t oz lb st
Température: c f k   (°C, fahrenheit, kelvin… acceptés)
Devises   : codes ISO sur trois lettres (USD, EUR, JPY, GBP…)
```

Exemples :

```
conv 10 km mi
conv 80 kg lb
conv 100 c f
conv 50 usd eur      # nécessite une connexion
```

## Organisation

```
main.go       CLI : lecture des trois arguments, dispatch, affichage
conv.go       table des unités, conversions linéaires et températures
currency.go   appel à l'API de taux, décodage, application du taux
```
