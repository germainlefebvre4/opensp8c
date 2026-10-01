# Proposal

## Why

Dans le sous-onglet « Documentation » de la vue Specs, les diagrammes Mermaid sont rendus en ligne, contraints par la largeur de la colonne de contenu. Les gros schémas (architecture, flux) restent trop petits pour que leurs libellés soient lisibles, et l'utilisateur n'a aucun moyen de les voir en grand sans quitter l'application.

## What Changes

- **Agrandissement au clic** : un clic sur un diagramme Mermaid de la page de documentation ouvre ce diagramme dans une fenêtre modale (lightbox) qui occupe l'essentiel de l'écran (environ 90 % de la largeur et de la hauteur disponibles).
- **Fermeture** : par la touche Échap, par un bouton de fermeture explicite, ou par un clic sur le fond de la modale.
- **Diagramme mis à l'échelle** : dans la modale, le diagramme est agrandi pour remplir l'espace disponible tout en conservant ses proportions (y compris les petits diagrammes, dont la taille naturelle est dépassée), sans zoom ni déplacement manuel dans cette première version.
- **Affordance et accessibilité** : le diagramme en ligne se comporte comme un bouton (curseur « zoom avant », focus clavier, activation par Entrée ou Espace) et porte un libellé accessible traduit en français et en anglais.
- **Repli inchangé** : un bloc Mermaid invalide continue d'être affiché comme bloc de code brut, non cliquable.

Hors périmètre :
- Zoom molette / boutons +/− et déplacement (pan) dans la modale : extension possible ultérieurement.
- Le sous-onglet « Spécifications » (rendu Markdown brut) et les autres rendus Markdown de l'application.
- Aucun changement backend ni de format des pages générées.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `spec-documentation-view`: ajout d'une exigence d'agrandissement d'un diagramme Mermaid au clic dans une modale. Les exigences existantes (rendu Markdown avec diagrammes Mermaid, repli sur bloc de code brut) ne changent pas.

## Impact

- Frontend uniquement : `frontend/src/components/MermaidDiagram.tsx` (conteneur cliquable et modale), tests associés, clés i18n `specs.json` fr/en.
- Dépendance existante `@radix-ui/react-dialog` réutilisée (déjà utilisée par `components/ui/ConfirmDialog.tsx`) ; aucune nouvelle dépendance.
- Chevauchement à coordonner avec le change en cours `docs-toc-wide-reading`, qui modifie aussi le style du conteneur `.mermaid-diagram` dans `MermaidDiagram.tsx` ; les deux changes portent sur des exigences distinctes de `spec-documentation-view`.
- Documentation : `docs/opensp8c/` n'est pas modifiée à la main (régénérée par la génération de documentation).
