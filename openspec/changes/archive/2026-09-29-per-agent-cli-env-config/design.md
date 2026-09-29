# Design

## Context

Voir `proposal.md` (Why / What Changes) pour la motivation. Points d'implémentation actuels utiles pour situer l'approche :

- `internal/preferences/preferences.go` définit `Preferences{ DefaultAgent, Env map[string]string, NativeQuestionMode, ... }`, persistée dans `preferences.json` via `Service.load()`/`save()`. `load()` contient déjà un précédent de migration ponctuelle (legacy `sessionAgents` → `sessions`), sans flag explicite : la migration se contente de vérifier une condition sur les données elles-mêmes, donc elle ne se rejoue jamais une fois les anciennes clés absentes.
- 5 points d'invocation de subprocess lisent aujourd'hui `customEnv = p.Env` avant d'appeler `session.StartSubprocess(..., agentCfg, ..., customEnv, ...)` : `internal/session/manager.go` (sessions nommées et anonymes, 2 occurrences), `internal/api/handlers/explore.go`, `internal/api/handlers/ff.go`, `internal/api/handlers/docs.go`, `internal/pool/worker.go`. Dans chacun, la variable portant l'`agents.AgentConfig` résolu (`resolved.config`, `cfg`, ou `agentCfg` selon le fichier) est déjà disponible au même endroit.
- `internal/api/handlers/preferences.go::GetPreferences` construit `systemEnv` en lisant en dur 3 variables d'environnement système (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) - ce sont les mêmes 3 clés que le frontend traite aujourd'hui comme "recommandées", bien qu'elles soient en réalité injectées dans tous les agents via le bag global `env`.
- `internal/agents/agents.go::SupportedAgents` est la liste canonique des 5 agents (`ID`, `Label`, `CLI`, `VersionArgs`), consommée à la fois par `agents.Detect`/`DetectAll` (backend) et par `GET /api/agents` → `useAgents()` (frontend, `AgentsRegistryTab`).
- Front : `frontend/src/pages/ConfigurationPage.tsx` a une table `AgentsRegistryTab` (lecture seule, un rendu par agent) suivie d'un formulaire `CliSettingsTab` unique portant 3 champs "recommandés" à traitement spécial (placeholder = valeur système, message hérité/surchargé) et une liste libre "Variables personnalisées" (clé/valeur, ajout/suppression). `SpecsPage.tsx` établit déjà le pattern "vue de détail pilotée par un paramètre de requête sur la même route" (`?selected=<name>`, miroir en `useState` via `useEffect`), sans route imbriquée dédiée - aucune page de l'app n'utilise de route imbriquée `/x/:id`.

## Goals / Non-Goals

**Goals:**
- Permettre d'ajouter des variables d'environnement propres à un agent CLI précis, injectées uniquement dans les subprocess de cet agent, aux 5 points d'invocation existants.
- Corriger la fuite actuelle des 3 variables Gemini vers tous les agents, via une migration ciblée et ponctuelle.
- Garder le bag global `env` intact dans son rôle actuel (s'applique à tous les agents), pour ne rien casser de ce qui existe.

**Non-Goals** (voir proposal.md - pistes notées, pas dans ce change) :
- Test de connexion / health-check d'un agent avec ses variables configurées.
- Sélection de modèle par liste de presets (au-delà du champ `GEMINI_MODEL` existant, simple texte libre).
- Tout mécanisme de sélection d'agent ou de modèle par colonne du Kanban.

## Decisions

### 1. `Preferences.AgentEnv map[string]map[string]string` + méthode de résolution `EnvFor(agentID)`
Nouveau champ `AgentEnv map[string]map[string]string \`json:"agentEnv,omitempty"\`` sur `Preferences`. `Service.load()` garantit, après chargement (fichier neuf ou existant), une entrée (potentiellement vide) pour chaque agent de `agents.SupportedAgents` - ainsi un futur agent ajouté à `SupportedAgents` obtient automatiquement une entrée vide au prochain chargement, sans migration dédiée.

Une méthode `func (p *Preferences) EnvFor(agentID string) map[string]string` retourne la fusion : copie de `p.Env`, puis écrasement par les entrées de `p.AgentEnv[agentID]` (clé par clé). Les 5 points d'invocation remplacent `customEnv = p.Env` par `customEnv = p.EnvFor(<agentID résolu>)` - un changement d'une ligne à chaque site, la variable portant l'ID étant déjà en portée partout.

Alternative rejetée : dupliquer la logique de fusion dans chacun des 5 call sites plutôt qu'une méthode partagée. Rejeté - la méthode centralise la règle de priorité (agent > global) à un seul endroit, testable isolément.

### 2. Migration ponctuelle des 3 clés Gemini connues, sans flag explicite
Dans `Service.load()`, après avoir garanti la présence de toutes les entrées `AgentEnv`, une passe de migration déplace `GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL` et `GEMINI_SANDBOX` de `p.Env` vers `p.AgentEnv["gemini"]` si elles sont présentes dans `p.Env`, puis les retire de `p.Env` et persiste (`s.save(&p)`, best-effort, même pattern que la migration `sessionAgents` existante). Aucun flag "migrated" nécessaire : la migration ne trouve plus rien à faire dès que ces clés ont quitté `p.Env`, donc elle ne se répète jamais et un utilisateur qui ajouterait volontairement une de ces clés au bag global plus tard ne serait pas re-migré à tort.

Alternative rejetée : un flag booléen `migratedAgentEnv` dans `preferences.json`. Rejeté - inutile ici (contrairement à un cas où la condition source pourrait redevenir vraie légitimement), et le précédent `sessionAgents` du même fichier n'en utilise pas non plus.

### 3. `PatchPreferences` : `agentEnv` remplace le dictionnaire d'un agent à la fois, sans toucher aux autres
Le corps de `PATCH /api/preferences` gagne un champ optionnel `agentEnv map[string]map[string]string`. Pour chaque clé d'agent présente dans le corps, `Service.SetAgentEnv` (nouvelle méthode : charge une fois, remplace `p.AgentEnv[agentID]` par le dictionnaire fourni pour chaque clé du corps, sauvegarde une fois) applique le même contrat que `SetEnv` aujourd'hui : la vue d'un agent envoie toujours l'intégralité de son propre dictionnaire résultant (pas un diff), les autres agents et le bag global ne sont pas affectés. Toute clé d'agent inconnue (absente de `agents.SupportedAgents`) est rejetée en 400, à l'image de la validation déjà faite sur `defaultAgent`.

### 4. Vue par agent pilotée par un paramètre de requête, pas une route imbriquée
`ConfigurationPage` suit le précédent déjà établi par `SpecsPage` (`?selected=`) : un clic sur une ligne du registre CLI positionne `?tab=cli&agent=<id>` sur la même route `/configuration` (pas de nouvelle route React Router), et l'onglet CLI bascule entre son contenu actuel (registre + formulaire global) et la vue dédiée à l'agent sélectionné. Un lien "retour" efface le paramètre `agent`.

Alternative rejetée : route imbriquée `/configuration/cli/:agentId`. Rejeté - aucune autre page de l'app n'utilise ce style de routage ; le paramètre de requête sur la même route donne le même bénéfice (lien profond, navigateur précédent/suivant) sans introduire un nouveau pattern.

### 5. Contenu de la vue par agent : liste libre partout, champs "recommandés" seulement pour Gemini
La vue dédiée à un agent affiche toujours la liste libre clé/valeur (même interaction que l'actuelle liste "Variables personnalisées", extraite en un petit composant partagé pour éviter la duplication) et un lien vers sa documentation officielle (`agents.AgentConfig.DocsURL`, nouveau champ, une URL statique par agent, exposée par `GET /api/agents` sous `docsUrl`). Seule la vue de Gemini affiche en plus les 3 champs "recommandés" avec leur traitement système/surcharge actuel (placeholder valeur système, message hérité/surchargé) - ce sont les seuls champs de ce type existants aujourd'hui ; les 4 autres agents n'ont que la liste libre.

## Risks / Trade-offs

- **Migration touche un fichier de préférences utilisateur réel** → Ciblée sur 3 clés connues uniquement, idempotente par construction (décision 2), couverte par un test qui vérifie qu'un `preferences.json` legacy avec ces clés dans `env` les retrouve dans `agentEnv.gemini` et plus dans `env` après un seul chargement.
- **Rupture de comportement pour un utilisateur qui dépendait (sciemment ou non) de la fuite actuelle des 3 clés Gemini vers un autre agent** → Accepté explicitement dans le proposal (**BREAKING**) : la fuite était un effet de bord non documenté, pas un comportement annoncé.
- **`agentEnv` avec des clés d'agent invalides si on contourne la validation du handler** → Mitigé par le rejet en 400 de toute clé absente de `agents.SupportedAgents`, à l'image de `defaultAgent`.

## Migration Plan

Migration en mémoire au premier `Service.load()` suivant le déploiement (décision 2), aucune étape manuelle requise. Déploiement en un seul lot frontend + backend (le frontend s'attend à recevoir `agentEnv` et `docsUrl` dans les réponses existantes ; aucun changement de forme cassant sur les champs déjà consommés). Rollback : revert du déploiement - un `preferences.json` déjà migré reste lisible par l'ancien code (le champ `agentEnv` est simplement ignoré, mais les 3 clés Gemini resteraient alors absentes de `env` jusqu'à ce que l'utilisateur les resaisisse ; à noter dans les release notes si un rollback est envisagé après migration réelle).
