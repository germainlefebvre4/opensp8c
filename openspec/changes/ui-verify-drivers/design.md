# Design

## Context

Voir `proposal.md` (Why). Ce change s'applique sur `verification-settings`, `verification-stage` et `ui-verify-step`, tous trois archivés. État observé du dépôt :

- `verification.Resolve(platform, workspace, change)` renvoie `Resolved{Conformity, UI, LaunchParams{UIStartCommand, UIBaseURL}}` ; les paramètres de lancement suivent la cascade workspace puis plateforme, champ par champ (`LaunchOverride` à pointeurs, `Level` = `Override` + `LaunchOverride`). Les types de `internal/preferences/verification.go` (`VerificationSettings`, `VerificationOverride`, `VerificationPatch`, `applyVerificationPatch`, `SetVerificationDefaults`) en sont le miroir persisté.
- `uiStep.runLocked` (`pool/verify_ui.go`) démarre l'application, prépare le dossier de preuves, puis appelle `runVerifierTurn` avec `uiVerifyDirective` et un tour construit par `buildUITurn`. `runVerifierTurn` résout `agentCfg` par `ResolveRoleConfig(ws, RoleVerifier)` et appelle `startSubprocessFn(procCtx, dir, agentCfg, directive, "", false, stderrLog, customEnv, false, langDirective)`, une variable de paquet que les tests remplacent.
- `agents.AgentConfig.BuildSubprocessArgs(base, extra)` ne produit pour Claude que `--print --verbose --input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt …` puis `--model` et `--effort`. Aucun paramètre de permission, de MCP ou de Chrome n'est passé : l'agent hérite de la config Claude de l'hôte.
- `runTurnText` (`pool/worker.go`) lit le flux de l'agent ligne à ligne (`classifyTurnLine`, `extractTextDelta`, `extractActivity`) sans exposer les événements `system`/`init` ni les appels d'outils.
- L'aide de la CLI installée (2.1.293) documente `--mcp-config`, `--strict-mcp-config`, `--allowedTools`, `--chrome`, `--no-chrome` et `--permission-prompts host|none` (« none : tout ce qui demanderait une permission est refusé automatiquement »). Le paquet `@playwright/mcp` (0.0.83 au moment de l'écriture) expose `--headless`, `--isolated` (profil en mémoire), `--output-dir` et `--viewport-size`.
- L'image Docker finale est un Alpine sans Node, sans navigateur et sans la CLI `claude` : les pilotes ne concernent que l'exécution sur un poste de développement.

## Goals / Non-Goals

**Goals:**
- Que l'utilisateur choisisse, par workspace, comment l'agent obtient un navigateur, avec un défaut (`auto`) strictement identique au comportement actuel.
- Un agent déterministe : jamais bloqué sur une permission, restreint aux outils prévus pour son pilote, échec explicite si le pilote n'est pas réellement disponible.
- Que `chrome` ne puisse pas être activé par un réglage global.

**Non-Goals:**
- Détecter ou choisir automatiquement un pilote, installer Node, Playwright ou ses navigateurs.
- Appliquer `--permission-prompts none` ou une liste d'outils à l'étape de conformité de `verification-stage`.
- Supporter un pilote avec un agent autre que Claude, ni empêcher techniquement un agent `chrome` malveillant de sortir de ses onglets (garde-fou par consigne seulement).
- Niveau change pour le pilote ou le guidage, ou fichier de guidage dans le dépôt.

## Decisions

### D1. La plateforme construit les paramètres de lancement selon le pilote ; l'agent choisit dans le périmètre du pilote

Cette décision remplace D1 de `ui-verify-step` (« l'agent choisit l'outil, la plateforme possède tout le reste ») : le reste demeure, mais le périmètre d'outils devient un réglage. `auto` conserve l'ancien comportement à l'identique. Un pilote explicite est restreint par `--allowedTools` aux outils de son serveur MCP plus `Read`, `Grep` et `Glob` (lecture seule du dépôt) : sans `Bash` ni `Edit`, la lecture seule n'est plus seulement une consigne, et le contrôle `git status` devient un second filet.

*Alternatives :* (a) un seul mode, Playwright imposé : ne couvre ni les projets qui ont un outillage propre, ni ceux qui préfèrent Chrome ; (b) laisser l'utilisateur écrire les flags : expose des options de sécurité sans garde-fou ; `custom` en donne le minimum utile (fichier MCP et outils).

### D2. `ExtraArgs` sur `AgentConfig`, signature de `StartSubprocess` inchangée

`AgentConfig` porte déjà des champs d'exécution fixés par lancement (`Model`, `Effort`, voir `preferences.ApplyRole`). On y ajoute `ExtraArgs []string`, que la branche Claude de `BuildSubprocessArgs` ajoute après `modelEffortArgs()`, et `SupportsDrivers() bool` (vrai pour `claude` seulement). `runVerifierTurn` copie `agentCfg`, y pose `ExtraArgs` puis appelle `startSubprocessFn` comme aujourd'hui.

*Alternative :* ajouter un paramètre à `StartSubprocess` et à `startSubprocessFn` : touche les huit sites d'appel et le simulateur des tests de pool pour rien. Le même mécanisme servira à l'étape de conformité si on décide plus tard de lui passer `--permission-prompts none`.

### D3. Réglages : types de `internal/verification` étendus, cascade par champ, `chrome` filtré à la résolution

`LaunchOverride` gagne `UIDriver *string`, `UIMcpConfig *string`, `UIAllowedTools *[]string` et `UIGuidance *string` ; `LaunchParams`/`Resolved` gagnent `UIDriver string`, `UIMcpConfig string`, `UIAllowedTools []string` et `UIGuidance string`. `Resolve` :
- pilote : `workspace > plateforme > auto` ; la valeur de plateforme `chrome` est écartée avant la cascade (`Level` de plateforme vs `Level` de workspace sont déjà deux appels distincts de `Resolve`) ;
- `uiMcpConfig`, `uiAllowedTools` : cascade champ par champ, la liste étant remplacée en entier (une liste vide ou sans entrée non vide vaut absente) ;
- `uiGuidance` : texte de plateforme puis texte de workspace rognés, joints par `"\n\n"`, vides ignorés.

La validation (`ParseDriver`, limite de 4000 caractères, 50 outils) vit dans le paquet `verification`, comme `ValidateBaseURL`. Le refus de `chrome` à la Configuration se fait dans `applyVerificationPatch` de `SetVerificationDefaults`, pas dans celle du workspace : les deux partagent la fonction, un paramètre `allowChrome` la distingue. Le filtre de résolution protège d'un `preferences.json` édité à la main sans faire échouer le chargement.

*Alternative :* ne refuser que dans l'interface. Écartée : l'API est le contrat, l'interface n'est qu'un client.

### D4. Plan de pilote pur et testable : `planDriver`

`pool/verify_driver.go` expose

```go
type driverPlan struct {
    Driver        string
    Args          []string   // ExtraArgs de l'agent
    AllowedTools  []string
    ToolPrefixes  []string   // appels comptant comme « usage du pilote »
    Servers       []string   // serveurs MCP attendus `connected`
    NeedChromeTools bool
    Directive     string     // phrase de consigne propre au pilote
    cleanup       func()
}
func planDriver(res verification.Resolved, artifactsDir string) (driverPlan, error)
```

Elle ne lance rien : elle vérifie ce qui ne dépend pas de l'agent (`exec.LookPath("npx")` pour `playwright`, fichier `uiMcpConfig` lisible et contenant `mcpServers` pour `custom`, `uiAllowedTools` non vide, type d'agent via `SupportsDrivers`), écrit la configuration MCP temporaire de `playwright` dans `os.MkdirTemp` (hors du worktree), et renvoie les paramètres. Elle est appelée au début de `runLocked`, avant `startApp` : un pilote impossible échoue avant d'avoir lancé l'application. Tous les cas d'échec se testent sans `claude`.

Paramètres par pilote (`Read Grep Glob` toujours ajoutés) :

| Pilote | `Args` | Outils autorisés | Serveurs attendus |
|---|---|---|---|
| `auto` | `--permission-prompts none` | (aucun ajout) | aucun |
| `playwright` | `--permission-prompts none --mcp-config <tmp> --strict-mcp-config --allowedTools …` | `mcp__playwright` | `playwright` |
| `chrome` | `--permission-prompts none --chrome --allowedTools …` | `mcp__claude-in-chrome` | (outils `mcp__claude-in-chrome__*`) |
| `custom` | `--permission-prompts none --mcp-config <uiMcpConfig> --strict-mcp-config --allowedTools …` | `uiAllowedTools` | clés de `mcpServers` du fichier |

La syntaxe exacte de `--allowedTools` pour « tous les outils d'un serveur MCP » (`mcp__playwright` ou `mcp__playwright__*`) est confirmée par la tâche 1.1 avant d'être figée dans le code.

### D5. Configuration Playwright MCP générée, version fixée, sortie dans le dossier de preuves

```json
{"mcpServers":{"playwright":{"command":"npx","args":["-y","@playwright/mcp@<version>","--headless","--isolated","--output-dir","<dossier de preuves>"]}}}
```

`--isolated` garde le profil en mémoire (aucune session, aucun cookie persistant), `--headless` supprime la dépendance à un écran, `--output-dir` fait atterrir les captures dans le dossier que le rapport liste déjà. La version est une constante du paquet `pool` (pas `latest`) : une mise à jour de l'outil ne doit pas changer silencieusement le comportement d'un agent autonome. Le fichier est supprimé par le `cleanup` du plan.

*Alternative :* `--user-data-dir` persistant pour garder une session de test connectée. Écartée : une session persistante fuit d'un change à l'autre ; l'authentification se décrit dans le guidage et les variables d'environnement.

### D6. Observation du flux : `turnTarget.observe`

`turnTarget` gagne `observe func(line []byte) error`, appelé par `runTurnText` pour chaque ligne lue (après `logRunRef`). Une erreur retournée interrompt le tour (même chemin que `turnError`, avec `procCancel`). `driverObserver` (dans `pool/verify_driver.go`) :
- sur l'événement `{"type":"system","subtype":"init",…}` lit `mcp_servers[]{name,status}` et `tools[]` ; échoue si un serveur attendu n'est pas `connected` (statut dans la raison), si le pilote `chrome` ne liste aucun outil `mcp__claude-in-chrome__*`, ou si le champ attendu est absent (fail-closed, raison « événement d'initialisation illisible ») ;
- sur les blocs `tool_use` des messages `assistant`, compte les appels dont le nom commence par un préfixe du pilote.

Après le tour, si le pilote n'est pas `auto` et que le verdict est `PASS` sans appel compté, l'étape échoue (« aucun outil du pilote n'a été utilisé »). L'événement `init` n'étant peut-être émis qu'à réception du premier tour, la vérification a lieu à sa réception et arrête l'agent avant tout appel d'outil plutôt qu'avant l'envoi du tour ; la tâche 1.1 constate le moment réel et ajuste si besoin.

*Alternative :* se contenter du verdict de l'agent. Écartée : un agent dont le serveur MCP n'a pas démarré peut conclure `PASS` sur la seule lecture du code ; le contrôle d'usage est le seul moyen indépendant de l'écarter.

### D7. `--permission-prompts none` pour tous les pilotes de l'étape `ui`

En `--print`, la CLI documente `host` comme défaut : un hôte SDK répond aux demandes, ce que `runTurnText` ne fait pas. `none` rend le comportement déterministe (refus immédiat, l'agent continue). Il est passé même en `auto` : cela ne retire aucune permission que l'hôte accorde déjà par ses règles, seulement l'attente. Comme la CLI évolue, la plateforme ne passe le flag que si `claude --help` le liste (détection mise en cache pour la durée du processus) ; sinon elle le journalise et poursuit. La tâche 1.1 constate sur la CLI réelle ce qui se passe sans le flag, avec et sans règle `allow`.

*Alternative :* `--permission-mode dontAsk` ou `bypassPermissions`. Écartées : la seconde donne trop de droits à un agent autonome, la première change la sémantique des règles de l'hôte.

### D8. Guidage dans le tour, pas dans la consigne système

`buildUITurn` reçoit le guidage résolu et ajoute, après la liste des scénarios et des tâches, un bloc « Indications de l'utilisateur » précisé comme indicatif : elles ne dispensent ni de la lecture seule ni du contrat de verdict, tous deux dans la consigne système. Placé dans le tour, le texte de l'utilisateur ne peut pas réécrire la consigne de la plateforme. La consigne dépend du pilote : `uiVerifyDirective` devient `buildUIDirective(plan)`, dont la phrase « Choose the way you drive the browser… » est conservée pour `auto` et remplacée pour les autres par « Drive the browser only with the tools of <pilote>; run no script and no command ». Pour `chrome` s'ajoute la consigne sur les onglets. Le texte du guidage n'est jamais journalisé séparément (il l'est dans le tour, comme le reste).

### D9. Rapport : `driver` et outils autorisés dans le marqueur de fin de run

Le marqueur `verify_run_end` de l'étape `ui` gagne `driver` et `allowed_tools` ; le lecteur du rapport (`GET …/verification/report`) les restitue, absents pour un ancien run ou une autre étape. Le bandeau du DetailPanel affiche un libellé court par pilote, avec une mention pour `chrome`.

### D10. Garde-fous `chrome` : opt-in par workspace, consigne, exclusivité, honnêteté sur les limites

`chrome` n'est pas isolé : l'agent agit dans un navigateur réel. Les garde-fous sont (a) l'opt-in réservé au workspace (D3), (b) l'avertissement dans l'écran de réglages, (c) la consigne de n'ouvrir que ses propres onglets, (d) le verrou exclusif existant qui garantit un seul agent à la fois. Ce sont des garde-fous de comportement, pas de sécurité : on ne les présente pas comme tels. Les captures prises par l'intégration Chrome ne sont pas garanties d'atterrir dans le dossier de preuves ; leur absence n'est pas une erreur.

*Alternative :* ne pas proposer `chrome` du tout. Écartée : c'est le choix demandé ; le risque est contenu par l'opt-in, l'avertissement et la visibilité du pilote dans chaque rapport.

## Risks / Trade-offs

- [Le premier lancement de `playwright` télécharge `@playwright/mcp` (réseau, quelques dizaines de secondes) et peut manquer de navigateur installé] → l'échec est explicite : serveur non `connected` à l'`init`, ou erreur d'outil rapportée par l'agent ; le texte d'aide indique `npx playwright install chromium`. Le délai maximal d'étape existant borne l'attente.
- [La forme de l'événement `init` ou de ses champs change avec la CLI] → fail-closed avec la raison « événement d'initialisation illisible » plutôt qu'un `PASS` aveugle ; un test sur un exemple réel capturé en 1.1 verrouille la forme attendue.
- [Un appel d'outil du pilote qui échoue compte comme « usage »] → limite connue : le contrôle écarte seulement le `PASS` sans aucune tentative ; la consigne demande de conclure `FAIL` si le navigateur ne répond pas et le rapport reste lisible.
- [`--permission-prompts none` absent d'une CLI plus ancienne] → détection par `claude --help`, l'étape continue sans le flag et le journalise.
- [Secrets saisis dans `uiGuidance`, stockés en clair dans `preferences.json` et envoyés au modèle] → aide du champ et documentation recommandent des variables d'environnement ; aucune détection automatique de secret.
- [`chrome` pilote un navigateur connecté] → voir D10 ; la mention visible dans le rapport rend l'usage auditable.
- [`uiAllowedTools` trop large en `custom` (par exemple `Bash`)] → la responsabilité est celle de l'utilisateur, qui a choisi `custom` ; le contrôle `git status` reste actif.
- [Un pilote est inopérant avec un agent non Claude] → échec immédiat et explicite plutôt qu'un pilote ignoré.

## Migration Plan

Aucune migration : un `preferences.json` sans les nouveaux champs équivaut à `auto` sans guidage, et `auto` ne change que par l'ajout de `--permission-prompts none` (D7). Retour arrière : repasser `uiDriver` à `auto` (ou retirer le champ) ; aucune donnée n'est écrite hors de `preferences.json` et de marqueurs de run ignorés par les anciens lecteurs.
