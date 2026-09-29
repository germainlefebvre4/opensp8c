# Proposal

## Why

Aujourd'hui, les variables d'environnement configurées dans Configuration > CLI (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`, et toute variable personnalisée) vivent dans un unique dictionnaire global (`preferences.json.env`), injecté tel quel dans le subprocess de **n'importe quel** agent CLI lancé par la plateforme (session interactive, exploration, fast-forward, worker de pool). Les 3 champs recommandés sont présentés dans l'UI comme dédiés à Gemini, mais mécaniquement ils fuient vers Claude, Codex, Antigravity et Copilot dès qu'ils sont renseignés. Il n'existe aucun moyen de déclarer une variable qui ne s'applique qu'à un agent CLI précis.

## What Changes

- Ajouter un second niveau de configuration d'environnement par agent (`agentEnv`), qui vient se superposer au dictionnaire global existant (`env`) : les variables globales restent injectées dans tous les agents (comportement inchangé), les variables spécifiques à un agent ne sont injectées que dans les subprocess de cet agent et l'emportent sur une variable globale de même nom.
- Migrer une fois, au premier chargement suivant la mise à jour, les 3 clés connues `GOOGLE_CLOUD_PROJECT`/`GEMINI_MODEL`/`GEMINI_SANDBOX` si elles sont présentes dans le dictionnaire global vers la configuration spécifique de Gemini, et les retirer du dictionnaire global. **BREAKING** (comportement) : après cette migration, ces 3 variables ne sont plus injectées dans les agents autres que Gemini, alors qu'elles l'étaient auparavant (effet de bord non intentionnel corrigé).
- Rendre chaque ligne du tableau Configuration > CLI cliquable pour ouvrir une vue dédiée à cet agent (même page, même route, sélection pilotée par un paramètre de requête `?agent=<id>`, sur le modèle déjà utilisé par Specs pour `?selected=`), affichant :
  - la liste librement éditable des variables d'environnement propres à cet agent (ajout/suppression clé-valeur, même interaction que l'actuelle liste "Variables personnalisées"), persistées et appliquées uniquement à cet agent ;
  - un lien vers la documentation officielle de cet agent CLI.
- Exposer et appliquer ces variables spécifiques à chacun des 5 points d'invocation de subprocess d'agent existants (sessions nommées, sessions anonymes, fast-forward, génération de docs, workers de pool), sans changer leur signature ni leur comportement au-delà de la fusion env global + env par agent.

Hors périmètre de ce change (pistes notées pour une exploration séparée) : test de connexion/health-check par agent, sélection de modèle avec liste de presets, sélection d'agent CLI ou de modèle par colonne du Kanban.

## Capabilities

### Modified Capabilities
- `agent-selection`: l'injection de variables d'environnement personnalisées passe d'un dictionnaire global unique à un dictionnaire global superposé à une configuration spécifique par agent, avec priorité à la configuration spécifique en cas de collision de clé.
- `platform-configuration`: Configuration > CLI gagne une vue par agent (accessible en cliquant sur sa ligne dans le registre), affichant ses variables d'environnement propres et un lien vers sa documentation officielle.

## Impact

- Backend : `internal/preferences/preferences.go` (nouveau champ `AgentEnv`, migration au chargement), `internal/api/handlers/preferences.go` (lecture/écriture de `agentEnv`), `internal/agents/agents.go` (nouveau champ `DocsURL` sur `AgentConfig`), les 5 points d'invocation de subprocess (`internal/session/manager.go` x2, `internal/api/handlers/explore.go`, `internal/api/handlers/ff.go`, `internal/api/handlers/docs.go`, `internal/pool/worker.go`) qui résolvent désormais `customEnv` par fusion global + par agent.
- Frontend : `frontend/src/pages/ConfigurationPage.tsx` (lignes du tableau CLI cliquables, nouvelle vue par agent pilotée par `?agent=`), `frontend/src/hooks/useAgentPreferences.ts`/`frontend/src/lib/api.ts` (type `Preferences.agentEnv`, `AgentStatus.docsUrl`), locales `frontend/src/locales/{fr,en}/configuration.json`.
- Aucune migration de données destructive : la migration des 3 clés Gemini connues est une réécriture ciblée de `preferences.json`, appliquée une seule fois et de façon idempotente.
