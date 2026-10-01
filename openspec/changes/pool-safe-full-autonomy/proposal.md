# Proposal

## Why

En mode `full-autonomy`, la seule barrière avant l'intégration du travail d'un agent dans la branche de l'utilisateur est la commande de validation, et deux défauts la rendent trompeuse :

- **Validation partielle silencieuse.** L'auto-détection ne retient `npm test` que si `node_modules` existe. Un worktree git neuf n'en contient jamais (le dossier n'est pas versionné) : pour un dépôt Go + frontend, seul `go test ./...` s'exécute, avec une simple ligne de log, et le worker peut fusionner automatiquement un frontend cassé. La spec actuelle (scénario « Projet Node sans dépendances installées ») entérine ce comportement.
- **Fusion sans garantie sur la cible.** Le worktree part du HEAD du moment du provisionnement, mais `MergeInto` fusionne dans la branche extraite au moment de la fin. Si l'utilisateur change de branche, le travail atterrit dans une branche non prévue ; s'il committe sur la branche d'origine, le résultat fusionné (travail + commits de l'utilisateur) n'a jamais été validé.

Le cas d'usage visé (full-autonomy, un worker, agent Claude ou Antigravity) fait de ces deux défauts le principal risque de régression silencieuse.

## What Changes

- **Auto-détection : un projet Node détecté mais non validable met le worker en pause.** Si un `package.json` déclare un script `test` et que `node_modules` est absent, l'auto-détection ne l'ignore plus : le worker passe à `paused` immédiatement (erreur d'environnement de validation, sans tour de guérison ni tentative consommée), avec une raison nommant les répertoires non validés et invitant à configurer une commande de validation (par exemple `npm ci && npm test`). Une commande configurée reste prioritaire et n'est pas concernée. Cela vaut quel que soit le mode de délégation, puisque la validation sert aussi à la revue.
- **Branche de base mémorisée.** Au provisionnement d'une nouvelle branche `feature/<change>`, le backend enregistre le nom de la branche d'origine dans la configuration git locale du dépôt (`branch.feature/<change>.opensp8c-base`). L'enregistrement suit la branche (supprimé avec elle) et survit aux pauses et redémarrages.
- **Refus de fusion hors de la branche de base.** Avant la fusion en full-autonomy, si la branche courante diffère de la branche de base enregistrée (ou si le HEAD est détaché), le worker ne fusionne pas, conserve branche et worktree, et passe à `paused` avec une raison nommant les deux branches. Une branche sans enregistrement (créée avant ce changement) n'est pas soumise à ce contrôle.
- **Intégration et revalidation avant fusion.** Avant de prendre le verrou de fusion, si la branche cible contient des commits absents de `feature/<change>`, le worker les intègre dans le worktree (`git merge` de la cible dans la branche du changement), puis rejoue la validation et, au besoin, la boucle d'auto-guérison sur ce résultat. Un conflit d'intégration est annulé et met le worker en pause. La fusion finale dans la cible porte ainsi sur un état déjà validé. Si la cible a encore avancé entre l'intégration et la fusion, le worker se met en pause plutôt que de fusionner un état non validé.
- **Documentation** : la section Agent Pool du README et `docs/opensp8c/architecture.md` décrivent ces comportements et les prérequis d'un usage en full-autonomy.

Hors périmètre :
- Les gardes d'annulation avant commit et merge, et le comportement de `Unlaunch --force` : ils relèvent du change `pool-cancel-before-merge`, indépendant de celui-ci.
- Le sandboxing de l'agent et l'héritage de la configuration de permissions de l'utilisateur.
- L'hygiène des tests et le versionnement de l'état runtime (`preferences.json`, `activity/`).
- Installer automatiquement les dépendances Node dans le worktree.

## Capabilities

### New Capabilities

Aucune.

### Modified Capabilities

- `agent-pool-orchestrator` :
  - `Commande de validation résolue par workspace` : une détection Node sans `node_modules` n'est plus ignorée mais signalée comme non validable.
  - `Pause immédiate sur erreur d'environnement de validation` : ajoute ce cas à la liste des erreurs d'environnement.
  - Deux nouvelles exigences : `Branche de base du changement` (enregistrement et refus de fusion hors base) et `Intégration et revalidation avant fusion`. Elles sont ajoutées, et non portées par `Fusion sûre en mode full-autonomy`, parce que le change `pool-cancel-before-merge` modifie cette exigence ; deux `MODIFIED` du même bloc s'écraseraient à l'archivage.

## Impact

- Backend : `backend/internal/pool/validation.go` (détection avec projets ignorés, erreur d'environnement), `worktree.go` (enregistrement et lecture de la base, intégration, test d'ascendance), `worker.go` (contrôle de base, étape d'intégration, factorisation validation + guérison pour être rejouée), `types.go` si une erreur typée est ajoutée.
- Chevauchement à coordonner avec `pool-cancel-before-merge` : les deux touchent le bloc full-autonomy de `worker.go` ; les tâches de ce change ne modifient que `worktree.go` pour la fusion elle-même (erreur typée) et ajoutent des cas à l'ordre des pauses existant.
- Comportement visible : les dépôts Node sans commande de validation configurée passent de « validation partielle silencieuse » à « pause explicite » ; c'est volontaire et c'est le seul changement de comportement pour les workspaces déjà configurés (aucun effet s'ils ont une commande de validation).
- Tests : `validation_test.go`, `worktree_test.go`, `finalize_test.go`. Aucune nouvelle dépendance, aucun changement de format de données persistées côté application.
