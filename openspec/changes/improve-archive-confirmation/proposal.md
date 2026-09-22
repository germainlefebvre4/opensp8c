# Proposal

## Why

Aujourd'hui, cliquer sur "Sync & Archive" (carte Kanban colonne Done) ou "Archiver" (DetailPanel) déclenche l'archivage immédiatement, sans confirmation, et la carte disparaît d'un coup dès que le serveur répond, sans aucun message de succès. L'action est pourtant irréversible (déplacement du dossier de change + réécriture des specs actives). Ce manque de confirmation et de feedback rend l'expérience abrupte et risque de provoquer des archivages accidentels.

## What Changes

- Ajout d'un composant `ConfirmDialog` générique (basé sur `@radix-ui/react-dialog` pour l'accessibilité : Escape, focus trap, `aria-modal`), affiché avant de déclencher l'archivage, quel que soit le point d'entrée (carte Kanban ou DetailPanel).
- Le dialog affiche le nom du change, rappelle que l'action synchronise les specs et archive le change de façon irréversible, et propose "Annuler" / "Archiver" (bouton destructif).
- Pendant l'appel API (inchangé : un seul appel `POST .../archive`), le dialog affiche un état de progression ("Synchronisation et archivage en cours...") avec spinner. En cas d'erreur, le dialog reste ouvert et affiche le message d'erreur inline avec un bouton "Réessayer", au lieu de fermer silencieusement.
- Ajout d'un système de toast générique réutilisable (provider + hook, ex. `@radix-ui/react-toast`) pour afficher un message de succès ("Change « <nom> » archivé") après la fermeture du dialog.
- Unification du libellé du bouton déclencheur entre la carte Kanban ("Sync & Archive", actuellement en dur, non traduit) et le DetailPanel ("Archiver", traduit) : un seul libellé, passé en i18n, utilisé aux deux endroits.
- Aucun changement du comportement backend ni de la commande CLI invoquée (`openspec archive <name> --yes` reste un appel unique) : la progression affichée est indicative, pas un suivi réel d'étapes séparées.

## Capabilities

### New Capabilities

(aucune)

### Modified Capabilities

- `change-archive`: le déclenchement de l'archivage depuis la carte Kanban passe par une confirmation explicite avant l'appel API, affiche un état de progression pendant l'appel, et un feedback de succès (toast) après l'archivage réussi, au lieu de disparaître silencieusement.
- `kanban-change-detail`: le clic sur "Archiver" dans le DetailPanel ouvre désormais le même dialog de confirmation partagé, avec le même feedback de progression et de succès que la carte Kanban, au lieu d'archiver immédiatement.

## Impact

- Frontend : nouveaux composants `ConfirmDialog` et système de toast (provider + hook), modifications de `ChangeCard.tsx` et `DetailPanel.tsx` pour utiliser ces composants au lieu d'appeler `useArchive` directement au clic, ajout de labels i18n, ajout des dépendances `@radix-ui/react-dialog` et `@radix-ui/react-toast` au `package.json`.
- Backend : aucun changement (l'endpoint `POST /api/workspaces/{id}/changes/{name}/archive` et le handler `archive.go` restent inchangés).
- Aucun changement de schéma de données ni d'API.
