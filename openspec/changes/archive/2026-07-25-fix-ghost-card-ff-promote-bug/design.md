## Context

L'application permet d'explorer de nouvelles idées sous forme d'une session anonyme (ghost card), puis de promouvoir cette exploration en un vrai change en cliquant sur le bouton de création dans le Kanban. Cette promotion démarre en tâche de fond une exécution de Fast-Forward (FF).
Cependant, l'agent utilisé pour ce Fast-Forward était codé en dur sur `"claude"` à la fois dans `runPromoteFF` et `TriggerFF`, ce qui échouait chez les utilisateurs non connectés à Claude Code. De plus, le subprocess gérant l'intégration de Gemini forçait l'injection systématique de `/opsx:explore ` en préfixe de chaque commande utilisateur, détruisant l'instruction de Fast-Forward `/opsx:ff`.

## Goals / Non-Goals

**Goals:**
- Respecter l'agent par défaut de l'utilisateur (`defaultAgent` configuré dans `preferences.json`) pour les Fast-Forward et promotions.
- Permettre à Gemini d'exécuter des slash commands sans les réécrire en `/opsx:explore`.

## Decisions

### Decision 1 : Ajout de ResolveAgentConfig dans `session.Manager`

Pour éviter de dupliquer la logique complexe de résolution de l'agent (gestion des préférences d'agent par défaut, session d'exploration, et fallbacks), nous exposons une méthode `ResolveAgentConfig` réutilisable sur le `session.Manager` :
```go
func (m *Manager) ResolveAgentConfig(workspaceID, changeName string) agents.AgentConfig
```
Et nous l'utilisons pour initialiser `cfg` dans `runPromoteFF` et `TriggerFF` de manière transparente et robuste.

### Decision 2 : Bypass du préfixe `/opsx:explore` pour toutes les slash commands de Gemini

Le subprocess Gemini réécrivait la commande utilisateur pour injecter le skill d'exploration s'il ne commençait pas par `/opsx:explore`. Nous généralisons cette règle pour autoriser n'importe quelle commande commençant par un slash (`/`) à contourner l'injection de préfixe.

## Risks / Trade-offs

- **Risk :** Risque de régression si un utilisateur entre du texte brut commençant accidentellement par `/` sans vouloir invoquer un skill.
  - *Mitigation :* Très faible probabilité car l'usage de `/` est standardisé pour les commandes système dans la CLI.
