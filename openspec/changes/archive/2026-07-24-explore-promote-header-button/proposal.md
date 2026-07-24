## Why

Permettre à l'utilisateur de formaliser et de convertir une session d'exploration anonyme (ghost card) en un change réel directement depuis le volet d'exploration (Explore panel). Cela évite de forcer l'utilisateur à retourner sur le Kanban pour effectuer un drag-and-drop manuel ou à taper une commande textuelle comme `/opsx:ff`.

## What Changes

- **Bouton d'action dans le Header** : Ajout d'un bouton d'action discret "Créer le change" dans le header du composant d'exploration anonyme (`ExploreAnonymousPanel`).
- **Comportement Responsive** : Le bouton s'adapte à la largeur du volet d'exploration : affichage du texte + icône si l'espace le permet, ou repli vers une icône seule (`✨`) avec un tooltip au survol si le volet est trop étroit.
- **Dialogue de Confirmation** : Le clic sur le bouton ouvre un dialogue de confirmation permettant de valider ou de personnaliser le nom du change (kebab-case) avant de lancer la promotion.
- **Déclenchement de la Promotion** : La validation du dialogue appelle l'API existante `/api/workspaces/{id}/explorations/{ghostId}/promote` et ferme le volet d'exploration.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `exploration-promote-to-change` : Ajout d'un point d'entrée alternatif pour déclencher la promotion d'une exploration via un bouton direct dans le volet d'exploration anonyme, avec dialogue de validation et comportement responsive.

## Impact

- **Frontend** :
  - `frontend/src/components/ExploreAnonymousPanel.tsx` : Ajout du bouton responsive dans le header et intégration du dialogue de confirmation.
  - `frontend/src/pages/KanbanPage.tsx` : Gestion coordonnée de la fermeture du volet et du passage de la carte en mode FF running.
  - `frontend/src/locales/fr/explore.json` / `en/explore.json` : Ajout des traductions pour le bouton et ses tooltips.
