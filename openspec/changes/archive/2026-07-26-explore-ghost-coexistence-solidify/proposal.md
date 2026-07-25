## Why

Lorsqu'un utilisateur promeut une session d'exploration anonyme en change réel (via le bouton "Créer le change" ou par drag-and-drop), le ghost card associé est actuellement supprimé immédiatement de la colonne "À explorer" et ses logs de conversation sont déplacés. Cela empêche l'utilisateur de revenir sur le chat d'exploration pour continuer à discuter avec l'IA et affiner la proposition si les tâches générées ne conviennent pas tout à fait. 

Permettre la coexistence temporaire du ghost card actif dans "À explorer" et d'une version "projet/brouillon" (unsolidified draft) du change dans "À faire" offre un flux d'allers-retours itératif et robuste, qui se consolide (se fige/se solidifie) dès que l'utilisateur commence à travailler sur le change ou choisit explicitement de figer la carte.

## What Changes

- Conserver le ghost card actif dans la colonne "À explorer" même après une promotion réussie (au lieu de le supprimer immédiatement).
- Faire apparaître le change promu dans la colonne "À faire" sous une forme "brouillon" visuellement distincte (grisée, bordure pointillée, badge projet/draft) tant que l'exploration associée est toujours active.
- Ajouter un bouton d'action "Figer" (Solidify) sur la carte de change en statut "brouillon" et dans son panneau de détails, permettant d'abandonner/fermer l'exploration et de rendre le change standard ("solide").
- Consolider/figer automatiquement le change brouillon si l'utilisateur coche une tâche, si le change passe à "In Progress" (en-cours), ou s'il est archivé.
- Copier l'historique de conversation de l'exploration vers le change lors de la promotion, tout en conservant l'original actif pour continuer les discussions d'exploration.

## Capabilities

### New Capabilities

### Modified Capabilities
- `explore-ghost-card`: Conserver la carte de l'exploration active après promotion et gérer sa suppression lors de la solidification du change.
- `exploration-promote-to-change`: Copier les logs d'exploration plutôt que de les déplacer immédiatement, et ne plus supprimer automatiquement le ghost record à la fin de la promotion.
- `kanban-board`: Permettre le rendu visuel distinct des changes "brouillons" (liés à un ghost actif) dans la colonne "todo", avec option de solidification explicite ou automatique.

## Impact

- **Backend (Go) :** 
  - Modification de `runPromoteFF` dans `backend/internal/api/handlers/explore.go` pour copier l'historique et ne plus supprimer le ghost record immédiatement.
  - Déclenchement automatique de la suppression du ghost associé lorsqu'une tâche d'un change est modifiée ou que son statut passe à `in-progress`.
- **Frontend (React/TypeScript) :**
  - Modification de `ChangeCard.tsx` pour détecter s'il existe un ghost portant le même nom et appliquer le style brouillon avec le bouton "Figer".
  - Ajout du bouton "Figer" dans le panneau de détails `DetailPanel.tsx` et gestion de l'appel d'API `deleteGhost` pour figer le change.
