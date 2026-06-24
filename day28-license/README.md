# day28 — license

Ajouter un en-tête de licence à la main, fichier par fichier, c'est la corvée
qu'on repousse jusqu'à l'oublier. `license` le pose partout d'un coup, dans le
bon style de commentaire selon le langage, et — surtout — sans jamais le doubler
si on relance.

```
license -name "Alice Martin" .            # MIT, année courante, partout
license -license apache-2.0 -name "Acme"  # une autre licence
license -n -name "Alice" .                # montre sans écrire
license -text NOTICE.txt .                # en-tête maison
```

## Décisions qui ont compté

- **Idempotent par construction.** C'est le piège évident : relancer l'outil et
  empiler trois fois le même en-tête. On cherche le marqueur `SPDX-License-Identifier`
  en haut du fichier ; s'il y est, on passe. On peut donc l'intégrer à un hook ou
  une CI sans crainte — il ne fait du bruit que sur les fichiers neufs.
- **Le SPDX comme pierre angulaire.** Une ligne `SPDX-License-Identifier: MIT`
  vaut un pavé juridique : standard, lisible par GitHub et les outils de
  conformité, et parfaite comme sentinelle d'idempotence. L'en-tête par défaut
  tient en deux lignes — copyright + SPDX — pas un roman que personne ne lit.
- **Le shebang reste roi.** Sur un script `#!/usr/bin/env bash`, coller l'en-tête
  en première ligne casse l'exécutable. On détecte le shebang et on glisse le bloc
  juste en dessous. Cas testé, parce que c'est exactement celui qu'on oublie.
- **Le style de commentaire suit le langage.** `//` pour Go/JS/Rust, `#` pour
  Python/shell/YAML : une `map[extension]prefix`. Extension absente = on ne sait
  pas commenter proprement, donc on ne touche pas. Mieux vaut sauter un fichier
  que d'y injecter une syntaxe invalide.
- **`-n` avant de se lancer.** Réécrire en masse, ça fait peur à raison. Le mode
  `-n` liste exactement ce qui changerait sans rien toucher — on regarde, puis on
  relance sans le drapeau. Comme [todoscan](../day26-todoscan) et
  [cloc](../day27-cloc), on saute `.git`, `node_modules` et les dossiers cachés.
- **On préserve les permissions.** Réécrire un fichier ne doit pas le faire passer
  en `0644` s'il était exécutable. On relit le mode d'origine et on le réapplique.

## Ce que j'ai laissé tomber

- **Le texte complet de la licence.** On pose un en-tête court (copyright + SPDX),
  pas les 21 lignes de la MIT dans chaque fichier. Le texte intégral, c'est le rôle
  d'un fichier `LICENSE` à la racine — un en-tête par fichier doit rester léger.
- **La mise à jour de l'année.** Un en-tête déjà posé en 2025 ne sera pas repassé
  en 2026 : on détecte la *présence*, pas la fraîcheur. Réécrire les années serait
  un autre mode, plus intrusif, à activer exprès.
- **Le retrait ou le remplacement d'en-tête.** L'outil ajoute, il ne désinstalle
  pas. Changer de licence proprement (retirer l'ancien bloc, poser le nouveau),
  c'est un travail à part entière — et risqué à automatiser à l'aveugle.
- **La détection des blocs `/* … */`.** On n'écrit que des commentaires de ligne,
  même là où un bloc serait plus idiomatique. C'est uniforme, ça se retire d'un
  `grep -v` au besoin, et ça évite de gérer l'ouverture/fermeture par langage.
