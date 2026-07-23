## Context

Actuellement, l'agent de code est hardcodé sur Claude à trois niveaux :
- Backend : `subprocess.go:31` → `exec.CommandContext(ctx, "claude", args...)`
- Frontend : `assistantName = 'Claude'` dans ExplorePanel, ExploreAnonymousPanel, TypingBubble
- Aucune persistance de préférence d'agent n'existe

Le modèle subprocess actuel (Claude CLI en stream-json) reste le socle — on l'ouvre à d'autres CLIs. L'historique de conversation est géré nativement par chaque CLI agent, pas par le backend.

Les sessions nommées (named) expirent après 30 min d'inactivité et le subprocess est tué. Pour qu'une session retrouve son agent après expiration, la préférence doit être persistée.

## Goals / Non-Goals

**Goals:**
- Permettre à l'utilisateur de choisir parmi plusieurs agents CLI (Claude, Codex, Gemini, Antigravity v2, Copilot)
- Détecter automatiquement quels agents sont installés et leur version
- Verrouiller l'agent d'une session à sa création (immuable)
- Persister la préférence globale et la mémoire des named sessions dans un fichier local
- Afficher l'agent actif dans l'en-tête des conversations

**Non-Goals:**
- Intégration API directe (REST/gRPC) vers les providers — CLI uniquement
- Préférence par workspace
- Changement d'agent en cours de session
- Migration des sessions existantes (Claude par défaut)
- Gestion des credentials / clés API des agents

## Decisions

### 1. CLI router au lieu d'API wrappers

**Décision** : On paramètre uniquement la commande subprocess, on ne crée pas d'adaptateurs API.

**Pourquoi** : Claude CLI gère l'historique de conversation via `--input-format stream-json`. Les autres agents CLI (codex, gemini, copilot) ont des interfaces similaires. Évite de gérer le format de streaming de chaque provider REST.

**Alternative écartée** : Appels API REST directs → nécessiterait la gestion des credentials, des formats de réponse différents, et une refonte majeure du système de messages.

### 2. Fichier preferences.json pour la persistance

**Décision** : Un seul fichier JSON à côté du fichier de config existant (`CONFIG_PATH`).

```json
{
  "defaultAgent": "claude",
  "sessionAgents": {
    "workspace-uuid/change-name": "codex"
  }
}
```

**Pourquoi** : Stockage suffisant pour deux clés (préférence globale + map de sessions). SQLite serait surdimensionné. Le fichier est lisible/debuggable directement. Aucun fichier projet modifié.

**Alternative écartée** : SQLite → overkill, dépendance supplémentaire.

**Alternative écartée** : Champ dans le Change YAML → modifie les fichiers projet, hors contraintes.

### 3. Verrouillage de l'agent à la création de session

**Décision** : L'agent est résolu une fois à la création du subprocess et ne change plus.

**Ordre de résolution** :
1. Pour une named session : `sessionAgents[workspaceID/changeName]` si existant
2. Sinon : `defaultAgent`
3. Si le CLI n'est plus installé : fallback Claude + warning

**Pourquoi** : L'historique de conversation est interne au subprocess CLI. Changer d'agent en cours de session casserait la cohérence du contexte.

### 4. Endpoint de détection des agents

**Décision** : `GET /api/agents` probe chaque CLI au moment de la requête (pas de cache long).

```
Pour chaque agent configuré :
  1. `which <cli>` → installed: bool
  2. `<cli> --version` → version: string | null
```

**Pourquoi** : L'état d'installation peut changer entre sessions. Un probe léger à la demande est plus fiable qu'un cache au démarrage du serveur.

### 5. Configuration statique des agents supportés

**Décision** : Les agents sont définis dans le code backend (pas dynamiques / pas configurables via UI).

```go
var SupportedAgents = []AgentConfig{
    {ID: "claude",      Label: "Claude",         CLI: "claude",   VersionArg: "--version"},
    {ID: "codex",       Label: "Codex",           CLI: "codex",    VersionArg: "--version"},
    {ID: "gemini",      Label: "Gemini",          CLI: "gemini",   VersionArg: "--version"},
    {ID: "antigravity", Label: "Antigravity",  CLI: "agy", VersionArg: "--version"},
    {ID: "copilot",     Label: "Copilot",         CLI: "gh",       VersionArg: "copilot --version"},
}
```

**Pourquoi** : Liste stable à court terme, pas besoin de dynamisme.

## Risks / Trade-offs

- **Interfaces CLI hétérogènes** → Les arguments acceptés par chaque CLI peuvent différer de Claude's `--input-format stream-json`. Il faudra valider les args pour chaque agent au moment de l'intégration. Mitigation : tester chaque CLI individuellement avant d'exposer dans l'UI.

- **Copilot via `gh` CLI** → Copilot n'a pas son propre binaire, il passe par `gh copilot`. La commande de version est `gh copilot --version`, pas `gh --version`. Cas spécial à gérer dans la config.

- **Antigravity CLI** → Interface inconnue à ce stade. Le nom du binaire et les arguments sont à confirmer. Mitigation : structure `AgentConfig` extensible, peut être ajouté progressivement.

- **Concurrent writes sur preferences.json** → Si plusieurs workspaces sont ouverts simultanément, des écritures concurrentes sont possibles. Mitigation : mutex en lecture/écriture dans le package preferences.

## Migration Plan

Aucune migration de données nécessaire.
- Sessions existantes : Claude par défaut (pas de `sessionAgents` dans preferences.json → résolution sur `defaultAgent = "claude"`)
- Si preferences.json absent : créé automatiquement avec `defaultAgent: "claude"` au premier usage
- Rollback : supprimer preferences.json + revenir au code antérieur
