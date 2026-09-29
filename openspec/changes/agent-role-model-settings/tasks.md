# Tasks

## 1. Registre d'agents : capacités modèle et effort

- [ ] 1.1 Confirmer que le change `explore-session-resume-and-restart` est terminé et fusionné avant de toucher `preferences.go` et `subprocess.go` (vérifier avec `openspec list` et `git status`)
- [ ] 1.2 Étendre `agents.AgentConfig` avec `ModelFlag`, `EffortFlag`, `EffortLevels`, `SeedModels`, `ListModels` et les champs d'exécution `Model`/`Effort`, et renseigner claude (`--model`, `--effort`, `low`..`max` dont `xhigh`, alias `fable|opus|sonnet|haiku`), agy (`--model`, `--effort` `low|medium|high|max`), gemini (`-m`, alias `auto|pro|flash|flash-lite`), codex (`-m`) ; copilot sans flag déclaré. Vérifier avec un test de table sur `agents.SupportedAgents`
- [ ] 1.3 Faire ajouter à `BuildSubprocessArgs` les flags de modèle puis d'effort uniquement quand la valeur et le flag sont non vides, pour chaque agent. Vérifier par des tests dans `agents_test.go` : claude avec les deux flags, codex avec modèle sans effort, valeur vide sans flag, copilot sans flag
- [ ] 1.4 Ajouter la découverte dynamique `agy models` (délai de 3 s, parsing `id<TAB>libellé`, fusion sans doublon avec la liste amorcée, cache mémoire de quelques minutes, repli sur la liste amorcée). Vérifier par des tests avec un exécutable factice : succès, échec, délai dépassé
- [ ] 1.5 Vérifier dans la documentation officielle les identifiants amorcés de Codex et les valider dans `SeedModels`. Vérifier que les listes amorcées de chaque agent sont couvertes par le test de 1.2

## 2. Préférences : schéma, résolution et validation

- [ ] 2.1 Ajouter à `Preferences` les champs `agentSettings`, `poolDefaults` et `workspaces` (voir design D1), tous `omitempty`, avec des pointeurs pour les champs partiels du pool. Vérifier qu'un `preferences.json` existant sans ces champs se charge sans erreur et sans réécriture (test dans `preferences_test.go`)
- [ ] 2.2 Implémenter `ResolveRole(workspaceID, role, lockedAgent)` pure et tolérante au receveur `nil` (design D2), avec les préréglages Claude. Vérifier par des tests de table : cascade workspace puis Configuration, rôle avant global, agent verrouillé, héritage bloqué quand l'agent change, effort hors niveaux écarté, préréglages Claude et absence de préréglage pour un autre agent
- [ ] 2.3 Implémenter la validation d'une mise à jour de réglages (agent supporté, rôle connu, effort dans les niveaux de l'agent défini au même niveau, modèle sans tiret initial ni espace ni caractère de contrôle, longueur maximale) et les méthodes `Set…` correspondantes globales et par workspace, avec réinitialisation par valeur nulle. Vérifier que chaque rejet laisse le fichier inchangé
- [ ] 2.4 Implémenter la résolution de la configuration de pool (`ResolvePool(workspaceID)` : workspace, sinon défaut global, sinon 3 / `hitl-review` / 3) avec validation de `size` entre 1 et 5. Vérifier par tests, y compris les valeurs invalides
- [ ] 2.5 Remplacer `EnvFor` par `EnvForWorkspace(workspaceID, agentID)` avec l'ordre global, agent, workspace, agent du workspace, en gardant `EnvFor` comme alias. Vérifier par les tests d'ordre et d'isolation entre workspaces

## 3. Application des rôles aux points de lancement

- [ ] 3.1 Ajouter `Role` (`explorer`, `ff`, `implementer`, `fixer`, `documenter`) et une fonction `ApplyRole` qui retourne l'`AgentConfig` avec `Model` et `Effort` résolus, sans changer la signature de `session.StartSubprocess`. Vérifier par un test que les arguments produits contiennent les flags attendus
- [ ] 3.2 Appliquer `explorer` dans `session/manager.go` (sessions nommées et anonymes) en respectant le verrou d'agent de session, et passer l'identifiant du workspace à `EnvForWorkspace`. Vérifier par des tests : verrou prioritaire sur l'agent, modèle et effort résolus pour l'agent verrouillé, session existante non modifiée par un changement de réglage
- [ ] 3.3 Appliquer `ff` dans `handlers/ff.go` et dans la génération d'artefacts de `handlers/explore.go`, et `documenter` dans `handlers/docs.go`, avec l'environnement du workspace. Vérifier par des tests de handlers qui inspectent la configuration passée au subprocess
- [ ] 3.4 Appliquer `implementer` dans `pool/worker.go` (un seul subprocess pour l'application initiale et l'auto-guérison) et ajouter `Worker.Role` pour qu'un worker relancé après corrections démarre avec `fixer`. Vérifier par des tests de worker : l'auto-guérison ne redémarre pas de subprocess et garde le modèle de l'implementer ; un worker créé avec `RoleFixer` applique les réglages du rôle `fixer`
- [ ] 3.5 Faire compléter `pool.Manager.Start` avec la configuration résolue quand la requête omet des champs (la requête explicite prime, sans persister). Vérifier par des tests de manager : résolution sans requête, requête explicite prioritaire, valeurs persistées inchangées

## 4. API

- [ ] 4.1 Étendre `GET`/`PATCH /api/preferences` avec `agentSettings` et `poolDefaults` (validation avant toute écriture, un rejet ne modifie rien). Vérifier par des tests dans `preferences_test.go` des handlers : lecture, enregistrement d'un rôle, rejets (agent inconnu, modèle `--flag`, rôle inconnu, effort hors niveaux, taille 0)
- [ ] 4.2 Ajouter `GET`/`PATCH /api/workspaces/{id}/settings` retournant `{ overrides, inherited, resolved }`, avec fusion partielle, `null` pour réinitialiser et 404 pour un workspace inconnu. Vérifier par des tests de handler : surcharge, réinitialisation, isolation entre workspaces, section d'un workspace retiré ignorée
- [ ] 4.3 Ajouter `GET /api/agents/models` (modèles avec source, niveaux d'effort, `supportsModel`, `supportsEffort` par agent). Vérifier par un test de handler couvrant un agent avec liste dynamique, un agent avec liste amorcée seule et copilot sans flag
- [ ] 4.4 Enregistrer les routes dans `router.go` et documenter les endpoints dans la documentation d'architecture du dépôt (`docs/opensp8c/`). Vérifier avec `cd backend && go build ./... && go test ./...` et une relecture de la documentation

## 5. Frontend : couche de données et composants partagés

- [ ] 5.1 Ajouter les types et hooks (`useAgentModels`, réglages globaux via `useAgentPreferences`, `useWorkspaceSettings` avec `usePatchWorkspaceSettings`) dans `frontend/src/lib/api.ts` et `frontend/src/hooks/`, avec invalidation des requêtes concernées. Vérifier par des tests de hooks (Vitest) sur la fusion des réponses et l'invalidation
- [ ] 5.2 Créer le composant `ModelField` (champ avec suggestions issues du catalogue, saisie libre, désactivé avec explication quand `supportsModel` est faux) et le sélecteur d'effort (masqué quand `supportsEffort` est faux). Vérifier par des tests de composant : saisie libre enregistrée, effort masqué pour gemini, champ désactivé pour copilot
- [ ] 5.3 Créer `RoleSettingsTable` (ligne globale plus cinq rôles, colonne du Kanban correspondante, note de portée du rôle `fixer`) paramétré par `scope` `global` ou `workspace`, avec valeur héritée, marquage des surcharges et bouton de réinitialisation en portée workspace. Vérifier par des tests : préréglages affichés comme défauts, surcharge marquée, réinitialisation, note du `fixer`
- [ ] 5.4 Créer `AgentPoolSettingsForm` (taille de 1 à 5, mode de délégation, tentatives) paramétré par `scope`, avec les mêmes règles d'héritage et de réinitialisation. Vérifier par des tests : valeur invalide refusée côté interface, héritage affiché en portée workspace
- [ ] 5.5 Ajouter les clés de traduction `en` et `fr` (`configuration.json`, `settings.json`, `dialogs.json` si besoin) pour tous les nouveaux libellés, dont « Colonnes » / « Columns » et « Environnement » / « Environment ». Vérifier qu'aucune clé n'est manquante entre `en` et `fr` (test de parité des locales existant ou ajouté)

## 6. Frontend : Configuration et Settings

- [ ] 6.1 Ajouter à `ConfigurationPage` le sous-onglet `columns` (`?tab=columns`) avec `RoleSettingsTable` en portée `global`, et les défauts du pool au-dessus de la vue de visibilité dans l'onglet Agent Pool. Vérifier par des tests dans `ConfigurationPage.test.tsx` : présence du sous-onglet, préréglages affichés, vue de visibilité conservée, enregistrement des défauts
- [ ] 6.2 Passer `workspaceId` à `SettingsPage` depuis `AppRoutes` (état vide « aucun workspace » sinon) et le structurer en sous-onglets `?tab=` : Agent Pool, Colonnes, Environnement, Spécialisations. Vérifier par des tests de page : état vide, changement de workspace, URL porteuse du workspace
- [ ] 6.3 Brancher les sous-onglets Agent Pool et Colonnes de Settings sur `AgentPoolSettingsForm` et `RoleSettingsTable` en portée `workspace`. Vérifier par des tests : valeur héritée visible, surcharge enregistrée pour le seul workspace actif, réinitialisation
- [ ] 6.4 Implémenter le sous-onglet Environnement (variables globales et par agent du workspace, ajout et retrait) et déplacer le contenu actuel de `SettingsPage` dans le sous-onglet Spécialisations sans changement de comportement. Vérifier par des tests : ajout et retrait d'une variable, les tests existants de spécialisations passent toujours
- [ ] 6.5 Pré-remplir `AgentPoolModal` avec la configuration résolue du workspace (`resolved.pool`), les ajustements de la modale n'étant appliqués qu'au lancement en cours. Vérifier par des tests de modale : pré-remplissage avec surcharge, ajustement ponctuel sans modifier la surcharge, workspace sans surcharge

## 7. Vérifications d'intégration

- [ ] 7.1 Lancer `cd backend && go vet ./... && go test ./...` puis `cd frontend && npm run lint && npm test && npm run build` et vérifier que tout passe
- [ ] 7.2 Valider le change avec `openspec validate agent-role-model-settings --strict` et vérifier l'absence d'erreur
- [ ] 7.3 Vérifier manuellement dans l'application (`make dev`) : régler un modèle de rôle dans Configuration puis le surcharger dans Settings d'un workspace, lancer une exploration et un pool sur ce workspace et contrôler les arguments passés au CLI (journal ou processus), vérifier qu'un autre workspace garde la valeur de Configuration, et qu'une session d'exploration verrouillée garde son agent
