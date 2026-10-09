# Proposal

## Why

La plateforme prévoit deux étapes de vérification automatique d'un change avant sa finalisation : une vérification de conformité (tour `/opsx:verify`) et une vérification UI (parcours dans l'application lancée). Ces étapes consomment beaucoup de tokens : elles doivent être activables à la demande, au bon niveau (toute la plateforme, un workspace, un seul change), et désactivées par défaut. Ce change pose uniquement le modèle de configuration et ses écrans, pour que les changes `verification-stage` et `ui-verify-step` puissent s'appuyer sur une valeur résolue sans réinventer l'héritage. Il ne vérifie rien lui-même.

## What Changes

- **Deux interrupteurs indépendants** : `conformity` (conformité code / specs / tâches) et `ui` (parcours UI). Chacun vaut `on`, `off` ou `inherit` (absence de valeur) à chaque niveau. Le niveau le plus spécifique qui n'est pas `inherit` l'emporte ; la valeur intégrée est `off`.
- **Trois niveaux** : Configuration (plateforme) dans `preferences.json`, workspace dans sa section de `preferences.json`, change dans son `.openspec.yaml` (lu dans le dépôt principal, jamais dans le worktree).
- **Paramètres de lancement de la vérification UI** : `uiStartCommand` et `uiBaseUrl`, définissables aux niveaux plateforme et workspace seulement (ils décrivent le projet, pas un change).
- **Écrans** : un sous-onglet « Vérification » dans Configuration et dans Settings, et un contrôle par change dans l'onglet Actions du DetailPanel, avec valeur héritée affichée, surcharge distinguée et réinitialisation, comme pour le pool.
- **API** : réglages plateforme via `GET/PATCH /api/preferences` (`verificationDefaults`) ; réglages workspace via `GET/PATCH /workspaces/{id}/settings` (`verification`) ; réglage de change via `PATCH /workspaces/{id}/changes/{name}/verification` ; le détail d'un change expose `verification` (surcharge et valeur résolue).
- **Nouveau rôle d'agent `verifier`** : réglable (agent, modèle, effort) avec la cascade existante ; sixième rôle, préréglage Claude `sonnet` / `medium`. Aucun subprocess n'est encore lancé avec ce rôle.

Hors périmètre : toute exécution de vérification, le marqueur persistant et la colonne `Verifying` (change `verification-stage`), le verrou, le lancement de l'application et les scénarios UI (change `ui-verify-step`), la politique d'échec (pause avec rapport, décidée dans `verification-stage`).

## Capabilities

### New Capabilities

- `verification-settings`: modèle `on` / `off` / `inherit` à trois niveaux pour la vérification de conformité et la vérification UI, valeur intégrée `off`, résolution, validation, persistance, paramètres de lancement UI, exposition API et écrans associés.

### Modified Capabilities

- `agent-role-settings`: un sixième rôle `verifier`, son préréglage Claude et la validation du rôle ciblé.
- `platform-configuration`: le sous-onglet « Colonnes » de Configuration liste six rôles (le sous-onglet « Vérification » est décrit dans `verification-settings`).
- `workspace-settings`: Settings passe à cinq sous-onglets avec « Vérification », et son sous-onglet « Colonnes » liste six rôles.
- `kanban-change-detail`: section Vérification dans l'onglet Actions du DetailPanel ; le détail d'un change expose ses réglages de vérification.

## Impact

- Backend : nouveau paquet `internal/verification` (types, résolution pure, validation), `internal/preferences` (`Preferences`, `WorkspacePrefs`, patchs, `roles.go`), `internal/openspec/change.go` (`openspecMeta`, `ChangeDetail`, écriture du réglage de change), `internal/api/handlers` (`workspace_settings.go`, handler de réglage de change, `preferences.go`), `internal/api/router.go`.
- Frontend : `lib/api.ts`, `lib/roleSettings.ts` (ou nouveau `lib/verification.ts`), nouveau formulaire partagé par `ConfigurationPage` et `SettingsPage`, `DetailPanel`, hooks (`useChangeDetail`, nouveau `useChangeVerification`), `RoleSettingsTable` (six rôles), locales fr/en (`configuration`, `settings`, `detailPanel`).
- Données : `preferences.json` (champs optionnels, rétrocompatible) et `.openspec.yaml` (champ optionnel `verification`). Les réécritures de `.openspec.yaml` par `SetLaunched` et `ClearKanbanState` passent par la struct `openspecMeta` : le nouveau champ doit y figurer, sinon il serait perdu à la première réécriture.
- API : champs optionnels en lecture et en écriture, rétrocompatibles. Aucune dépendance nouvelle.
