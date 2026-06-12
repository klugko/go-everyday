# day17 — md2html

Prendre un Markdown et en faire **une page HTML qui se suffit à elle-même** :
CSS intégré, rien à charger, prête à envoyer en pièce jointe ou à déposer sur
n'importe quel serveur.

## Le problème

On écrit tout en Markdown — notes, comptes rendus, petites docs. Mais le jour
où il faut l'envoyer à quelqu'un qui n'a pas d'éditeur qui le rend joliment,
on se retrouve à copier-coller dans un mail et tout casse. Je voulais un seul
binaire qui sorte un `.html` propre, lisible, et surtout **autonome** : pas de
feuille de style à côté, pas de CDN.

```
md2html notes.md > notes.html      # vers stdout, à rediriger
md2html -o notes.html notes.md     # ou directement dans un fichier
cat brouillon.md | md2html         # stdin marche aussi
```

## Décisions qui ont compté

- **Un parseur ligne à ligne, pas une lib.** Le Markdown qu'on écrit à la main
  tient dans une poignée de blocs : titres, paragraphes, listes, citations,
  code, séparateurs. Une boucle avec un `switch` sur la première ligne de
  chaque bloc suffit — chaque fonction de bloc consomme ses lignes et rend
  l'index suivant. Pas besoin d'un AST pour ça.
- **Le `code` inline traité avant tout le reste.** Rien ne doit être
  interprété entre backticks — ni le gras, ni les liens. On coupe donc la
  ligne sur les backticks : les segments impairs deviennent du `<code>`
  échappé, les autres passent par les regex de mise en forme. Un backtick
  orphelin est simplement rendu au texte.
- **Échapper d'abord, formater ensuite.** Tout passe par `html.EscapeString`
  avant les regex. Un `<script>` dans le Markdown s'affiche, il ne s'exécute
  pas. C'est un convertisseur, pas un interpréteur de HTML embarqué.
- **Le `<title>` vient du premier `# titre`.** À défaut, le nom du fichier ;
  à défaut, « Document ». Une page sans titre fait mauvais effet en pièce
  jointe.
- **Le CSS tient en trente lignes.** Largeur de colonne confortable, police
  système, fond gris sur le code, filet sur les citations. Le but est que ça
  se lise bien partout, pas de faire un thème.

## Ce que j'ai laissé tomber

- **Les listes imbriquées et les tableaux.** Ça double la complexité du
  parseur pour des cas rares dans une note qu'on envoie. Les listes plates
  couvrent l'essentiel.
- **Les images.** Une page « autonome » avec des `<img>` qui pointent vers des
  fichiers locaux ne l'est plus. Il faudrait les encoder en base64 dans la
  page — un autre projet.
- **La coloration syntaxique.** Le langage du bloc est gardé en
  `class="language-go"`, libre à qui veut de brancher highlight.js. Moi je
  m'arrête au fond gris.
- **Le HTML brut dans le Markdown.** Tout est échappé, point. C'est le prix
  d'une sortie qu'on peut envoyer sans relire.
