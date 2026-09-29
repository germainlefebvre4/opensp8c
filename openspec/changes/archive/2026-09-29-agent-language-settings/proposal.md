# Proposal

## Why

La plateforme lance des agents CLI (exploration, fast-forward, génération de documentation, workers du pool) sans jamais leur dire dans quelle langue s'exprimer : la locale de l'application n'existe que dans le navigateur (`localStorage.lang`), le backend n'en a aucune notion, et les prompts sont tous en anglais. Résultat : la langue des conversations, des artefacts OpenSpec, de la documentation et du code produit dépend du hasard (langue du dernier message, du contenu des specs, de l'habitude du modèle). Il faut un réglage explicite, à trois niveaux (chat, code, documentation), transmis aux agents.

## What Changes

- Ajouter dans Configuration > Langue un réglage de langue à trois niveaux :
  - **chat** : langue des échanges avec l'agent (exploration nommée et anonyme) ; défaut `auto` = locale de l'application ;
  - **documentation** : langue de la documentation générée **et des artefacts OpenSpec** (proposal, design, specs, tasks produits par fast-forward ou par promotion d'une exploration) ; défaut `auto` ;
  - **code** : langue de tout ce qui est écrit dans le code (identifiants, commentaires, messages de commit, messages d'erreur, textes visibles) ; défaut `en`.
- Persister ces trois réglages globalement dans `preferences.json` (objet `agentLanguages`), exposés via les endpoints existants `GET`/`PATCH /api/preferences`. Chaque niveau vaut `auto` (seulement pour chat et documentation) ou un code de langue explicite (`en`, `fr`, ...). La liste des langues proposables est une donnée unique, pas un cas codé en dur, pour accueillir de nouvelles langues sans refonte.
- Persister aussi la locale courante de l'UI côté backend (champ `uiLocale`, synchronisé par le frontend à chaque changement de langue) afin que le backend puisse résoudre `auto` au lancement, y compris pour les workers du pool qui tournent sans requête du navigateur.
- Construire, à chaque lancement d'agent, une consigne de langue selon le rôle (chat / documentation / code / rôles composites) et la transmettre à l'agent : en prompt système supplémentaire pour les agents qui l'acceptent (Claude), préfixée au message utilisateur pour ceux qui n'ont pas de flag de prompt système (Gemini, Antigravity ; Codex et Copilot à vérifier).
- La consigne est une **valeur par défaut, l'explicite gagne** : une demande explicite de l'utilisateur, le contenu des artefacts du change ou les conventions du projet (par exemple un i18n existant) l'emportent sur elle. Elle n'est jamais formulée comme une contrainte ferme.

Hors périmètre : traduction de l'interface elle-même (déjà couverte par `i18n-core`), traduction rétroactive de contenus existants, réglage de langue par workspace ou par change, langues de l'UI au-delà de EN/FR.

## Capabilities

### New Capabilities
- `agent-language-settings`: réglages de langue chat/code/documentation, leur persistance, leur résolution (`auto` = locale de l'app) et leur transmission aux agents lancés par la plateforme.

### Modified Capabilities
- `platform-configuration`: Configuration > Langue gagne, en plus du sélecteur de langue de l'UI, la section de réglage des langues de chat, de documentation et de code.
- `spec-documentation-generation`: la documentation générée est produite dans la langue « documentation » résolue, au lieu d'une langue non spécifiée.

## Impact

- Backend : `internal/preferences/preferences.go` (champs `AgentLanguages`, `UILocale`, setters, valeurs par défaut), `internal/api/handlers/preferences.go` (lecture/écriture/validation), un nouveau module de construction de la consigne de langue, `internal/session/subprocess.go` (point de passage `StartSubprocess`), et les sites de lancement : `internal/session/manager.go` (exploration nommée, anonyme), `internal/api/handlers/explore.go` (promotion), `ff.go`, `docs.go`, `internal/pool/worker.go`.
- Frontend : `frontend/src/lib/api.ts` (type `Preferences`), `frontend/src/hooks/useAgentPreferences.ts`, `frontend/src/i18n.ts` (synchronisation de la locale vers le backend), `frontend/src/pages/ConfigurationPage.tsx` (onglet Langue), locales `configuration.json` fr/en.
- Prérequis satisfaits : `move-configuration-to-sidebar` (onglet « Langue » de `ConfigurationPage.tsx`, sélection par `?tab=`) et `per-agent-cli-env-config` (`EnvFor` sur les sites de lancement) sont appliqués et archivés ; ce change s'appuie sur leur état actuel. Le delta `platform-configuration` complète l'exigence existante « Sous-onglet Langue dans Configuration » sans la modifier.
- Pas de migration destructive : les champs sont optionnels et absents des `preferences.json` existants (valeurs par défaut appliquées).
