# Tasks

## 1. Backend - liste de base et extensions utilisateur

- [x] 1.1 Créer `backend/internal/openspec/specializations.go` avec une constante `BaseAgentSpecializations []string` (frontend, backend, database, api-design, devops, testing, security, documentation, ux-design, data, mobile) et une fonction `CombineSpecializationVocabulary(custom []string) []string` qui fusionne la base et les extensions en dédupliquant (une valeur déjà dans la base n'est jamais dupliquée) ; vérifier avec un test unitaire couvrant le cas de doublon
- [x] 1.2 Ajouter `CustomAgentSpecializations []string` à `preferences.Preferences` (`backend/internal/preferences/preferences.go`) et une méthode `SetCustomAgentSpecializations([]string) error`, sur le modèle de `SetEnv` ; vérifier avec un test dans `backend/internal/preferences` (set puis reload)
- [x] 1.3 Étendre `PreferencesHandler.GetPreferences`/`PatchPreferences` (`backend/internal/api/handlers/preferences.go`) pour exposer/accepter `customAgentSpecializations` (même pattern que `env`) ; vérifier avec un test dans `backend/internal/api/handlers` (GET reflète un PATCH précédent)
- [x] 1.4 Ajouter un handler `GetAgentSpecializations` (nouveau fichier `backend/internal/api/handlers/specializations.go` ou ajout à `preferences.go`) exposant `GET /api/agent-specializations` → `{"base": [...], "custom": [...]}` en combinant `openspec.BaseAgentSpecializations` et les extensions lues via `prefsSvc`, et enregistrer la route dans `router.go` ; vérifier avec un test handler que la réponse contient bien les deux listes séparées

## 2. Backend - champ `agent_specialization` et dérivation LLM contrainte

- [x] 2.1 Ajouter `AgentSpecialization []string` (yaml `agent_specialization`) à `Tags` (`backend/internal/openspec/change.go`) ; vérifier que `go build ./...` compile et qu'un `.openspec.yaml` existant sans ce champ se parse toujours sans erreur (test de non-régression sur le parsing)
- [x] 2.2 Étendre le prompt et le parsing dans `LLMDeriveComplexityAndComponents` (`backend/internal/openspec/tagger.go`) pour aussi demander `agent_specialization` (liste de slugs choisis dans un vocabulaire fermé donné en contexte) ; extraire une fonction pure `FilterToVocabulary(values, vocabulary []string) []string` qui ne garde que les valeurs présentes dans le vocabulaire fermé, et l'appliquer au résultat LLM avant de l'assigner à `tags.AgentSpecialization` ; vérifier avec un test unitaire de `FilterToVocabulary` (valeur hors vocabulaire filtrée, valeur valide conservée, vocabulaire vide → résultat vide)
- [x] 2.3 Faire passer le vocabulaire fermé de spécialisation à travers `TagChange` (nouveau paramètre `specializationVocabulary []string` sur `TagChange(changeRoot, workspaceRoot string, forceRetag bool, specializationVocabulary []string) error`, `tagger.go`) et mettre à jour ses 3 points d'appel pour construire ce vocabulaire via `preferences.Service` + `CombineSpecializationVocabulary` avant l'appel :
  - `backend/cmd/server/main.go` (`tagUntaggedChanges`, batch au démarrage)
  - `backend/internal/api/handlers/archive.go` (trigger à l'archivage)
  - `backend/internal/api/handlers/tags.go` (`Retag`, endpoint manuel)
  Vérifier que `go build ./...` compile et que `go test ./...` passe sur les 3 packages modifiés
- [x] 2.4 Vérifier manuellement : sur un change de ce repo sans `agent_specialization` (ex. `kanban-delete-change`), appeler `POST /api/workspaces/{id}/changes/kanban-delete-change/retag` et confirmer dans `.openspec.yaml` que `agent_specialization` contient uniquement des valeurs de la liste de base (aucune extension définie par défaut)

## 3. Frontend - affichage sur la carte Kanban

- [x] 3.1 Ajouter `agent_specialization: string[]` à l'interface `Tags` (`frontend/src/hooks/useChanges.ts`)
- [x] 3.2 Dans `ChangeCard.tsx`, ajouter le rendu de badges `agent_specialization` dans le bloc `change.tags && (...)` existant (après les points de complexité, ~ligne 207), un badge par valeur, masqué si le tableau est vide
- [x] 3.3 Vérifier manuellement dans le navigateur : après le retag de la tâche 2.4, la carte du change affiche les badges de spécialisation aux côtés du badge `type` existant

## 4. Frontend - écran de configuration Settings

- [x] 4.1 Ajouter `customAgentSpecializations?: string[]` à l'interface `Preferences` (`frontend/src/lib/api.ts`) et une fonction `getAgentSpecializations()` (`GET /api/agent-specializations`)
- [x] 4.2 Ajouter un hook `useAgentSpecializations()` (react-query, `frontend/src/hooks/useAgentPreferences.ts` ou nouveau fichier dédié) sur le modèle de `usePreferences`/`usePatchPreferences`, réutilisant `usePatchPreferences` pour la mutation d'ajout/retrait (`customAgentSpecializations`)
- [x] 4.3 Créer `frontend/src/pages/SettingsPage.tsx` : section unique listant la liste de base en lecture seule, les tags personnalisés avec une action de suppression chacun, et un champ d'ajout validant le format kebab-case côté client avant l'appel `PATCH /api/preferences`
- [x] 4.4 Enregistrer la route `/settings` dans `App.tsx` (même pattern que `/agents`, cross-workspace)
- [x] 4.5 Ajouter l'entrée de navigation "Settings" dans `Layout.tsx` et la clé `settings` dans `frontend/src/locales/{en,fr}/navigation.json`
- [x] 4.6 Créer les fichiers `frontend/src/locales/{en,fr}/settings.json` (clés de l'écran : titre, liste de base, ajout, suppression, erreur de validation) et les enregistrer dans `frontend/src/i18n.ts` (import + `ns` + `resources`)
- [x] 4.7 Vérifier manuellement dans le navigateur : ouvrir Settings, ajouter un tag personnalisé, confirmer qu'il apparaît immédiatement dans la liste et qu'il est proposé par le tagger LLM lors d'un prochain retag (réutiliser la vérification de la tâche 2.4 avec ce nouveau tag)

## 5. Vérification de bout en bout

- [x] 5.1 Lancer `go test ./...` côté backend et `npm run build`/lint côté frontend, confirmer qu'aucune régression n'apparaît
- [x] 5.2 Scénario complet manuel : ajouter un tag personnalisé dans Settings, déclencher un retag sur un change existant, vérifier que `.openspec.yaml` contient `agent_specialization` avec au moins une valeur, que la carte Kanban affiche le badge correspondant, et qu'aucune valeur hors vocabulaire fermé n'apparaît même après plusieurs retags successifs
