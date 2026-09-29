# Tasks

## 1. Backend - modèle de données `agentEnv` et migration

- [x] 1.1 Ajouter `AgentEnv map[string]map[string]string \`json:"agentEnv,omitempty"\`` sur `Preferences` dans `backend/internal/preferences/preferences.go`, et vérifier `cd backend && go build ./...`
- [x] 1.2 Dans `Service.load()`, garantir après chargement (fichier neuf ou existant) une entrée (potentiellement vide) dans `p.AgentEnv` pour chaque agent de `agents.SupportedAgents`, et vérifier avec un test dans `backend/internal/preferences/preferences_test.go` qu'un `preferences.json` neuf comme un `preferences.json` existant sans `agentEnv` obtiennent bien une entrée vide par agent supporté après `Load()`
- [x] 1.3 Dans `Service.load()`, après la garantie de la tâche 1.2, migrer une fois `GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL` et `GEMINI_SANDBOX` de `p.Env` vers `p.AgentEnv["gemini"]` si présentes dans `p.Env` (les retirer de `p.Env`), persister le résultat, et vérifier avec un test qu'un `preferences.json` legacy contenant ces clés dans `env` les retrouve dans `agentEnv.gemini` et plus dans `env` après un seul appel à `Load()`, et qu'un second appel à `Load()` ne modifie plus rien (idempotence)
- [x] 1.4 Ajouter `func (p *Preferences) EnvFor(agentID string) map[string]string` dans `backend/internal/preferences/preferences.go` : copie de `p.Env` puis écrasement clé par clé par `p.AgentEnv[agentID]`, et vérifier avec un test que (a) sans entrée pour cet agent, le résultat égale `p.Env`, (b) une clé présente à la fois dans `p.Env` et `p.AgentEnv[agentID]` prend la valeur de `p.AgentEnv[agentID]`, (c) une clé propre à un autre agent n'apparaît pas dans le résultat
- [x] 1.5 Ajouter `func (s *Service) SetAgentEnv(updates map[string]map[string]string) error` dans `backend/internal/preferences/preferences.go` : charge une fois, remplace `p.AgentEnv[agentID]` par le dictionnaire fourni pour chaque clé d'agent présente dans `updates` (sans toucher aux agents absents de `updates` ni à `p.Env`), sauvegarde une fois, et vérifier avec un test que mettre à jour l'agent `gemini` ne modifie ni `p.Env` ni `p.AgentEnv["claude"]`

## 2. Backend - endpoint `/api/preferences` et registre des agents

- [x] 2.1 Ajouter `DocsURL string` sur `agents.AgentConfig` dans `backend/internal/agents/agents.go`, renseigner une URL de documentation officielle pour chacun des 5 agents de `SupportedAgents`, exposer ce champ dans `AgentStatus` (`docs_url` en JSON) peuplé par `Detect`, et vérifier avec un test dans `backend/internal/agents/agents_test.go` que `DetectAll()` retourne le bon `DocsURL` pour chaque agent
- [x] 2.2 Mettre à jour `PreferencesHandler.GetPreferences` dans `backend/internal/api/handlers/preferences.go` pour inclure `agentEnv` dans la réponse JSON (une clé par agent supporté, `{}` si vide), et mettre à jour `backend/internal/api/handlers/preferences_test.go` en conséquence
- [x] 2.3 Mettre à jour `PreferencesHandler.PatchPreferences` dans `backend/internal/api/handlers/preferences.go` pour accepter un champ optionnel `agentEnv map[string]map[string]string` dans le corps, rejeter en 400 toute clé d'agent absente de `agents.SupportedAgents` (même validation que `defaultAgent`), et appeler `Service.SetAgentEnv` sinon ; ajouter un test vérifiant la mise à jour réussie d'un agent connu et le rejet d'un identifiant d'agent inconnu

## 3. Backend - fusion de l'environnement aux points d'invocation de subprocess

- [x] 3.1 Remplacer `customEnv = p.Env` par `customEnv = p.EnvFor(resolved.config.ID)` dans les 2 occurrences de `backend/internal/session/manager.go` (sessions nommées et anonymes), et vérifier `cd backend && go test ./internal/session/...`
- [x] 3.2 Remplacer `customEnv = p.Env` par `customEnv = p.EnvFor(cfg.ID)` dans `backend/internal/api/handlers/explore.go` et `backend/internal/api/handlers/ff.go`, et `customEnv = p.EnvFor(cfg.ID)` dans `backend/internal/api/handlers/docs.go`, et vérifier `cd backend && go test ./internal/api/handlers/...`
- [x] 3.3 Remplacer `customEnv = p.Env` par `customEnv = p.EnvFor(agentCfg.ID)` dans `backend/internal/pool/worker.go`, et ajouter un test dans `backend/internal/pool/worker_test.go` vérifiant qu'un worker démarré avec une variable spécifique à l'agent configuré dans `agentEnv` la reçoit dans son subprocess, alors qu'un worker démarré pour un autre agent ne la reçoit pas
- [x] 3.4 Lancer `cd backend && go test ./...` : tous les tests passent

## 4. Frontend - types et lien de documentation

- [x] 4.1 Mettre à jour `frontend/src/lib/api.ts` : `Preferences` gagne `agentEnv: Record<string, Record<string, string>>`, `AgentStatus` gagne `docsUrl: string`
- [x] 4.2 Extraire de `CliSettingsTab` (`frontend/src/pages/ConfigurationPage.tsx`) un petit composant partagé `EnvVarList` (liste libre clé/valeur : ajout, suppression, édition), utilisé à la fois par la section "Variables personnalisées" existante (comportement inchangé) et par la nouvelle vue par agent (tâche 5.2) ; vérifier `cd frontend && npm run test` (tests existants de `CliSettingsTab` toujours au vert)

## 5. Frontend - vue de configuration dédiée par agent CLI

- [x] 5.1 Dans `frontend/src/pages/ConfigurationPage.tsx`, rendre chaque ligne du tableau `AgentsRegistryTab` cliquable pour positionner `?tab=cli&agent=<id>` via `useSearchParams` (même pattern que `SpecsPage`'s `?selected=`), sans changer de route
- [x] 5.2 Créer `AgentCliConfigView` dans `frontend/src/pages/ConfigurationPage.tsx` : affiche le libellé de l'agent, un lien de documentation (`docsUrl`), `EnvVarList` lié à `agentEnv[agentId]` (persisté via `PATCH /api/preferences` avec `{ agentEnv: { [agentId]: {...} } }`), et un lien de retour qui efface le paramètre `agent` ; pour l'agent `gemini` uniquement, affiche en plus les 3 champs recommandés (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) avec leur traitement système/surcharge actuel (repris de `CliSettingsTab`), retirés du formulaire global
- [x] 5.3 Brancher `AgentCliConfigView` dans le rendu de l'onglet CLI de `ConfigurationPage` : affichée quand `?agent=` est renseigné, contenu actuel (registre + formulaire global sans les 3 champs Gemini) sinon
- [x] 5.4 Mettre à jour `frontend/src/locales/{fr,en}/configuration.json` : libellés de la vue par agent (titre, lien de documentation, retour) et suppression des libellés des 3 champs recommandés du formulaire global (déplacés dans la vue Gemini)
- [x] 5.5 Ajouter des tests dans `frontend/src/pages/ConfigurationPage.test.tsx` (mock de `useAgents`/`usePreferences`/`usePatchPreferences`) vérifiant : le clic sur la ligne `codex` affiche sa vue avec une liste libre vide, l'ajout d'une variable dans la vue `gemini` n'affiche pas les variables d'un autre agent, la vue `gemini` affiche les 3 champs recommandés avec le traitement système/surcharge, la vue `claude` ne les affiche pas ; vérifier `cd frontend && npm run test`

## 6. Vérification d'intégration

- [x] 6.1 Lancer `cd backend && go test ./...` et `cd frontend && npm run test` : tous les tests passent
- [x] 6.2 Lancer l'application (`make dev`), ouvrir Configuration > CLI, cliquer sur la ligne Gemini, vérifier que les 3 champs recommandés s'y affichent (plus dans le formulaire global), ajouter une variable propre à Codex, relancer une session ou un worker utilisant Codex et vérifier manuellement (logs ou variable observable dans le subprocess) qu'elle est bien injectée, et qu'elle ne l'est pas pour un subprocess Claude
