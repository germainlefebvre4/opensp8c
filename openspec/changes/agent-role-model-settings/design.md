# Design

## Context

Voir `proposal.md` (Why) et les specs `agent-role-settings` et `workspace-settings` pour les exigences.

État actuel, vérifié dans le code :

- Tout subprocess d'agent naît de `session.StartSubprocess(ctx, path, agentCfg, ...)`, qui construit ses arguments via `AgentConfig.BuildSubprocessArgs`. Les points d'appel sont : `session/manager.go` (sessions nommées et anonymes), `handlers/explore.go` (génération d'artefacts depuis une exploration promue, qui correspond au rôle `ff`), `handlers/ff.go`, `handlers/docs.go` et `pool/worker.go`.
- L'agent est résolu par `session.Manager.resolveAgent` : verrou de session (`prefs.Sessions`) sinon `defaultAgent`. Aucun modèle ni effort n'est passé aux CLI.
- `Preferences` (`preferences.json`) est global. Les workspaces vivent dans `backend/config.yaml` avec un nom et un chemin ; leur identifiant est `workspace.StableID`.
- Chaque lancement charge déjà les préférences pour l'environnement (`EnvFor`) et la langue (`LanguageDirective`). Le rôle se résout au même endroit.
- `SettingsPage` n'a pas de `workspaceId`. `ConfigurationPage` gère ses sous-onglets via `?tab=` (`agent-pool`, `cli`, `language`).
- La configuration du pool arrive uniquement du corps de `POST /workspaces/{id}/pool/start` (`AgentPoolConfig`), sans persistance.
- Les CLI diffèrent : `claude` accepte `--model` et `--effort` (`low` à `max`, dont `xhigh`), `agy` accepte `--model`, `--effort` (`low|medium|high|max`) et `agy models`, `gemini` et `codex` acceptent `-m` et n'ont pas de flag d'effort. Pour `codex` et `copilot`, `BuildSubprocessArgs` retombe aujourd'hui sur les arguments de Claude « en attendant la validation de leur CLI » : ces deux agents ne sont donc pas réellement pilotables aujourd'hui.
- Le flux de review HITL (« Demander des corrections ») est spécifié dans `agent-pool-ui` mais n'a pas de point d'entrée backend : seul `MergeAndCleanup` existe dans `pool/worktree.go`.

## Goals / Non-Goals

**Goals:**
- Une résolution unique, pure et testable (agent, modèle, effort) pour un rôle et un workspace, utilisée par tous les points de lancement.
- Aucun changement de signature de `StartSubprocess`, pour ne pas entrer en conflit avec le change en cours sur `subprocess.go`.
- Des capacités d'agent (flags, niveaux d'effort, modèles) déclarées dans le registre `agents`, pas dans le frontend.
- Un composant d'interface partagé entre Configuration et Settings.

**Non-Goals:**
- Valider que le modèle saisi librement existe réellement chez le fournisseur.
- Rendre `codex` et `copilot` pilotables (adaptation de leurs arguments) : hors périmètre, leurs flags de modèle restent déclarés seulement si vérifiés.
- Implémenter le flux « Demander des corrections » de la review HITL.
- Surcharger la langue des agents par workspace (`agent-language-settings` reste global).
- Changer le modèle en cours de session ou de worker.

## Decisions

### D1. Schéma dans `preferences.json`

```
{
  "agentSettings": {                 // Configuration (global)
    "global": { "agent": "", "model": "", "effort": "" },
    "roles":  { "explorer": {…}, "ff": {…}, "implementer": {…}, "fixer": {…}, "documenter": {…} }
  },
  "poolDefaults": { "size": 3, "delegationMode": "hitl-review", "maxAttempts": 3 },
  "workspaces": {
    "<workspaceId>": {
      "agentSettings": { "global": {…}, "roles": {…} },
      "pool":     { "size": 4 },     // champs partiels
      "env":      { "FOO": "…" },
      "agentEnv": { "claude": { "FOO": "…" } }
    }
  }
}
```

Un champ vide ou absent signifie « hériter ». Les champs de pool utilisent des pointeurs (`*int`, `*string`) pour distinguer « non surchargé » d'une valeur légitime. Le fichier existant reste valide sans migration (`omitempty` partout). **Alternatives écartées** : un fichier par workspace dans le dépôt du workspace (rejeté par l'utilisateur, `preferences.json` retenu) ; ranger les rôles dans `config.yaml` (fichier édité à la main, non écrit par l'application).

### D2. Résolution pure dans `internal/preferences`

`(*Preferences).ResolveRole(workspaceID string, role Role, lockedAgent string) Resolved{Agent, Model, Effort string}`, sans I/O, tolérante au receveur `nil` comme `LanguageDirective`.

1. **Agent** : `lockedAgent` s'il est non vide ; sinon le premier défini parmi rôle@workspace, global@workspace, rôle@Configuration, global@Configuration ; sinon `defaultAgent`. Un agent non installé retombe sur `claude` par `resolveAgentFromID`, comme aujourd'hui.
2. **Modèle et effort**, chacun séparément : pour chaque niveau dans le même ordre, la valeur du niveau n'est retenue que si l'agent effectif de ce niveau égale l'agent résolu. L'agent effectif d'un niveau est son agent propre, sinon celui du premier niveau moins spécifique qui en définit un, sinon `defaultAgent`. Sinon, préréglage du rôle pour l'agent résolu (Claude seulement), sinon vide.
3. Un effort qui n'appartient pas aux `EffortLevels` de l'agent résolu est écarté silencieusement.

**Pourquoi** : c'est le seul moyen de garantir « un modèle Claude n'est jamais passé à Codex » sans rejeter des configurations valides individuellement. **Alternative écartée** : rejeter à l'enregistrement toute combinaison incohérente ; impossible car la cohérence dépend de niveaux modifiables indépendamment.

Point à relire : le préréglage vient après le réglage global, comme dans la cascade demandée (workspace avant Configuration, rôle avant global à niveau égal). Un utilisateur qui fixe un modèle global explicite l'emporte donc sur les préréglages de rôle (ex. l'explorateur n'aura plus `opus/high` de lui-même). L'interface le rend visible en affichant le préréglage comme valeur par défaut seulement quand le champ n'est défini nulle part.

### D3. Capacités déclarées dans le registre d'agents

`agents.AgentConfig` gagne `ModelFlag string`, `EffortFlag string`, `EffortLevels []string`, `SeedModels []Model` et `ListModels func(ctx) ([]Model, error)`, ainsi que deux champs d'exécution `Model` et `Effort`. Valeurs : claude (`--model`, `--effort`, `low..max` avec `xhigh`, alias `fable|opus|sonnet|haiku`), agy (`--model`, `--effort`, `low|medium|high|max`, `ListModels` via `agy models`), gemini (`-m`, pas d'effort, alias `auto|pro|flash|flash-lite`), codex (`-m`, pas d'effort, seed minimal vérifié à l'implémentation), copilot (aucun flag déclaré tant que son CLI n'est pas validé).

`BuildSubprocessArgs` ajoute `ModelFlag Model` puis `EffortFlag Effort` quand la valeur et le flag sont non vides. **Une fonction `WithRole(cfg, resolved) AgentConfig` retourne une copie de la config avec `Model` et `Effort`, sans toucher à la signature de `StartSubprocess`** : chaque appelant fait `cfg = preferences.ApplyRole(...)` juste avant. **Alternative écartée** : ajouter deux paramètres à `StartSubprocess` (déjà 10 paramètres, et fichier en cours de modification dans un autre change).

### D4. Catalogue de modèles

`GET /api/agents/models` retourne, par agent, `{ models: [{id, label, source: "seed"|"cli"}], effortLevels, supportsModel, supportsEffort }`. La liste amorcée est toujours retournée ; pour `agy`, la découverte lance `agy models` avec un délai de 3 s (même garde que `Detect`), fusionne le résultat sans doublon et retombe sur la liste amorcée en cas d'échec. Résultat mis en cache en mémoire quelques minutes pour éviter de lancer le CLI à chaque ouverture d'écran. La saisie libre est toujours acceptée : l'interface est un champ avec suggestions, pas un `select` fermé.

**Alternative écartée** : ne proposer que la découverte dynamique (seul `agy` sait lister) ; ne servirait aucun autre agent. **Alternative écartée** : une liste figée dans le frontend ; les identifiants de Codex et de Copilot changent trop vite.

### D5. API

- `GET`/`PATCH /api/preferences` (existant) étendu avec `agentSettings` et `poolDefaults`. Validation avant toute écriture, comme les langues d'agent : un rejet ne modifie rien.
- `GET /api/workspaces/{id}/settings` retourne `{ overrides, inherited, resolved }` pour l'affichage de l'héritage ; `PATCH /api/workspaces/{id}/settings` fusionne, `null` sur un champ le réinitialise (revient à l'héritage). Réponse 404 si le workspace n'existe pas.
- `POST /workspaces/{id}/pool/start` : les champs absents ou nuls du corps sont complétés par la configuration résolue ; les champs présents l'emportent.

**Alternative écartée** : tout faire passer par `PATCH /api/preferences` avec un objet `workspaces` : mélange l'identifiant d'un workspace inexistant et la validation métier, et oblige le client à renvoyer toute la section.

### D6. Environnement du workspace

`EnvFor(agentID)` devient `EnvForWorkspace(workspaceID, agentID)` : global, agent global, workspace, agent du workspace. `EnvFor` reste un alias sans workspace pour les appelants sans contexte. Les points de lancement passent l'identifiant du workspace qu'ils ont déjà.

### D7. Pool et rôle `fixer`

`pool.Manager.Start` complète la config depuis les préférences résolues. `runWorker` (implementer) résout le rôle une fois et l'applique au subprocess partagé de la boucle d'auto-guérison. Le rôle `fixer` est **résolu et testé, mais il n'a pas de point de lancement aujourd'hui** (voir Context). Le change expose `ResolveRole(..., RoleFixer, ...)` et l'applique dans le worker relancé avec `Worker.Role`, de sorte que l'implémentation future du flux de correction n'ait qu'à démarrer un worker avec `RoleFixer`. L'interface affiche déjà la portée du rôle.

### D8. Frontend

- `AppRoutes` passe `workspaceId` à `SettingsPage`, qui gagne des sous-onglets via `?tab=` (`agent-pool` par défaut, `columns`, `environment`, `specializations`), sur le modèle de `ConfigurationPage`.
- Deux composants partagés, paramétrés par une `scope` (`global` ou `workspace`) : `AgentPoolSettingsForm` et `RoleSettingsTable`. En portée `workspace`, ils affichent la valeur héritée, marquent les surcharges et proposent « réinitialiser ».
- Sélecteur de modèle : champ avec suggestions issues de `GET /api/agents/models`. Le champ effort est masqué quand `supportsEffort` est faux, et le champ modèle est désactivé avec une explication quand `supportsModel` est faux (copilot).
- `AgentPoolModal` se pré-remplit depuis `resolved.pool` de `GET /api/workspaces/{id}/settings`.
- Nouvelles clés dans les locales `en` et `fr` (`configuration.json`, `settings.json`).

## Risks / Trade-offs

- **[Le rôle `fixer` n'a aucun effet tant que le flux de correction n'existe pas]** → Documenté dans l'interface et dans la spec ; le seam est en place et testé, et un test vérifie que le worker relancé applique `RoleFixer`.
- **[Un modèle saisi librement peut être invalide]** → Le CLI échoue au lancement ; le worker passe en `paused` avec la raison, et une exploration remonte l'erreur. Validation de forme uniquement (pas de tiret initial, pas d'espace) pour empêcher l'injection de flags.
- **[`codex` et `copilot` ne sont pas réellement pilotables]** → Leur flag de modèle n'est déclaré que s'il est vérifié ; sinon le champ est désactivé plutôt que de casser le lancement.
- **[Le global explicite l'emporte sur les préréglages de rôle]** → Comportement à valider par l'utilisateur ; l'inverse (préréglage avant le global) rend le global inopérant pour Claude.
- **[Cache de découverte obsolète]** → Durée courte et repli sur la liste amorcée ; la saisie libre reste possible.
- **[Conflit avec `explore-session-resume-and-restart`]** → Aucun changement de signature ; les modifications de `preferences.go` sont additives (nouveaux champs et méthodes). À implémenter après le merge de ce change.
- **[`preferences.json` grossit avec des sections de workspaces retirés]** → Sections ignorées à l'exécution ; aucune purge automatique pour ne rien perdre en cas de réajout.

## Migration Plan

Aucune migration de données : tous les champs sont optionnels et l'absence équivaut au comportement précédent (défaut du CLI) augmenté des préréglages Claude. Pour l'utilisateur de Claude, le comportement change donc dès le déploiement : explorer passe à `opus/high`, ff et implementer à `sonnet/medium`, documenter à `haiku/low`. Retour arrière : déployer la version précédente ; les nouveaux champs de `preferences.json` sont ignorés.

## Open Questions

- Liste exacte des identifiants amorcés pour Codex, à vérifier dans la documentation au moment de l'implémentation.
- Le libellé exact du rôle `ff` dans l'interface (« Génération d'artefacts » ou « Fast-forward »).
