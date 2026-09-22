# Proposal

## Why

Le Kanban n'offre aujourd'hui aucun moyen de supprimer une carte (un change) dont on ne souhaite plus. Seuls les ghost cards (brouillons d'exploration éphémères, pas encore de dossier `openspec/changes/`) peuvent être supprimés. Une carte "solide" en To Do, In Progress, To Review ou Done ne peut qu'être archivée (Done) ou déplacée — elle reste indéfiniment sur le board si elle n'aboutit jamais à un archivage, ce qui accumule du clutter que l'utilisateur ne peut pas nettoyer lui-même.

## What Changes

- Nouvelle action **Supprimer** pour un change "solide", disponible en To Do, In Progress, To Review et Done (pas Archived, qui reste un historique en lecture seule). **BREAKING** au sens local : suppression définitive du dossier `openspec/changes/<name>/`, sans corbeille ni possibilité de restauration.
- La suppression est bloquée (bouton désactivé) si un worker de l'Agent Pool Orchestrator est actif sur ce change, pour éviter de laisser un worktree/process orphelin.
- Si le change a un ghost associé encore actif (exploration non "figée"), le ghost record correspondant est supprimé avec lui — pas d'orphelin visible dans la colonne To Explore.
- La suppression est protégée par une confirmation modale (nouveau `DeleteChangeDialog`, sur le modèle de `ResetTasksDialog` existant).
- Le `DetailPanel` gagne un nouvel onglet **Actions**, qui centralise :
  - Supprimer (nouveau)
  - Archiver (déplacé depuis le footer conditionnel actuel du statut Done)
  - "Approuver & Fusionner" / "Demander correction" (déplacés depuis le footer conditionnel actuel du statut To Review, tels quels — ces boutons restent sans handler, leur implémentation est hors périmètre)
- Le bouton "Sync & Archive" déjà présent au hover sur la carte (`ChangeCard.tsx`, statut Done) est conservé tel quel, en plus de sa présence dans l'onglet Actions : les deux surfaces coexistent.
- Nouvel endpoint backend `DELETE /api/workspaces/{id}/changes/{name}` qui supprime le dossier du change, refuse (409) si un worker actif y est attaché, et purge le ghost record associé le cas échéant.

## Capabilities

### New Capabilities

(aucune nouvelle capacité — la fonctionnalité s'intègre dans la capacité existante ci-dessous)

### Modified Capabilities

- `kanban-change-detail`: ajoute la suppression d'un change (requirement + endpoint), ajoute l'onglet Actions du DetailPanel qui héberge Supprimer, Archiver, et les boutons To Review, et modifie en conséquence l'emplacement des requirements existants "Archiver un change depuis le DetailPanel" (déplacé du footer vers l'onglet Actions).

## Impact

- **Frontend**: `frontend/src/components/DetailPanel.tsx` (nouvel onglet Actions, suppression des footers conditionnels Done/To Review au profit du tab), nouveau composant `DeleteChangeDialog.tsx` (sur le modèle de `ResetTasksDialog.tsx`), `frontend/src/lib/api.ts` (nouvel appel `deleteChange`), nouveau hook `useDeleteChange`.
- **Backend**: `backend/internal/api/handlers/kanban.go` (nouveau handler `DeleteChange`), `backend/internal/api/router.go` (nouvelle route `DELETE /workspaces/{id}/changes/{name}`), lecture de `pool.Manager.Status()` pour détecter un worker actif sur le change, réutilisation de la logique de suppression de ghost déjà présente dans `explore.go` (`DeleteGhost`) pour la cascade.
- **Pas d'impact** sur la colonne Archived ni sur le flux d'archivage lui-même (comportement inchangé, seulement déplacé dans l'UI).
