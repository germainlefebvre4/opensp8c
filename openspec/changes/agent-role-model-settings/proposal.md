# Proposal

## Why

Aujourd'hui tous les agents lancés par la plateforme (exploration, fast-forward, workers du pool, génération de documentation) utilisent le CLI par défaut du CLI choisi, sans aucun contrôle sur le modèle ni sur l'effort de raisonnement : le même réglage sert à réfléchir sur une exploration et à rédiger de la documentation, ce qui coûte cher ou sacrifie la qualité. De plus, l'écran Settings n'est lié à aucun workspace et le pool d'agents n'a aucune configuration persistée : chaque lancement repart de valeurs codées en dur dans la modale. Il faut pouvoir régler l'agent, le modèle et l'effort par rôle, globalement puis par workspace.

## What Changes

- Ajout d'un réglage **agent / modèle / effort** pour chacun des cinq rôles d'exécution : `explorer`, `ff`, `implementer`, `fixer`, `documenter`, plus un réglage global qui sert de base à tous les rôles. Chaque champ hérite de la valeur du niveau supérieur lorsqu'il est vide.
- Résolution en cascade : rôle du workspace → global du workspace → rôle de Configuration → global de Configuration → défaut du CLI (aucun flag passé). Le verrou d'agent par session d'exploration reste prioritaire sur l'agent ; modèle et effort sont alors ceux définis pour cet agent.
- Préréglages par défaut pour l'agent Claude : `explorer` (modèle de raisonnement, effort `high`), `ff`, `implementer` et `fixer` (modèle de code, effort `medium`), `documenter` (modèle de documentation, effort `low`). Pour les autres agents, aucun préréglage : le défaut du CLI s'applique.
- Le rôle `fixer` ne s'applique qu'au relancement d'un worker après « Demander des corrections » en revue HITL ; la boucle d'auto-guérison partage le subprocess de l'`implementer` et conserve son modèle.
- Catalogue de modèles par agent : liste amorcée côté backend (alias stables quand l'agent en propose), découverte dynamique lorsque le CLI sait lister ses modèles (`agy models`), et saisie libre possible pour tous les agents. Les niveaux d'effort sont déclarés par agent ; l'effort est masqué pour les agents qui n'en prennent pas.
- Configuration de l'Agent Pool persistée (taille, mode de délégation, tentatives max) : défauts globaux dans Configuration > Agent Pool, surcharge par workspace dans Settings > Agent Pool ; la modale de lancement du pool est pré-remplie avec la valeur résolue.
- Nouveau sous-onglet **Colonnes** (Columns en anglais) dans Configuration et dans Settings pour régler les cinq rôles.
- **Settings devient un écran propre au workspace actif** avec des sous-onglets : Agent Pool, Colonnes, Environnement (variables d'environnement globales et par agent qui surchargent celles de Configuration) et Spécialisations (contenu actuel de l'écran, inchangé).
- Stockage dans `preferences.json` : réglages globaux et section par workspace indexée par son identifiant. Extension de `GET`/`PATCH /api/preferences` (ou d'endpoints dédiés par workspace).
- Le passage des flags modèle et effort aux CLI s'appuie sur `--model`/`--effort` (Claude, Antigravity) et `-m` (Codex, Gemini) ; l'effort n'est transmis qu'aux agents qui le supportent.

## Capabilities

### New Capabilities
- `agent-role-settings`: rôles d'exécution des agents, réglage agent/modèle/effort par rôle avec héritage et préréglages, résolution en cascade global → workspace, catalogue de modèles et niveaux d'effort par agent, application au lancement de chaque type de subprocess.
- `workspace-settings`: écran Settings scopé au workspace actif, ses sous-onglets (Agent Pool, Colonnes, Environnement, Spécialisations) et le stockage des surcharges par workspace.

### Modified Capabilities
- `platform-configuration`: nouveau sous-onglet Colonnes et défauts de l'Agent Pool dans Configuration ; l'onglet Agent Pool n'est plus uniquement une vue de visibilité en lecture seule.
- `agent-pool-orchestrator`: la configuration du pool est persistée et résolue en cascade (global → workspace) au lieu d'être fournie uniquement à chaque lancement.
- `agent-pool-ui`: la modale de lancement du pool est pré-remplie avec la configuration résolue du workspace.
- `agent-specialization`: l'écran Settings n'est plus un écran plat global mais un écran scopé au workspace où les spécialisations forment un sous-onglet.
- `agent-selection`: le verrou d'agent par session reste prioritaire ; l'injection des variables d'environnement au démarrage superpose désormais les surcharges du workspace.
- `workspace-management`: Réglages n'est plus une page indépendante du workspace.

## Impact

- **Backend** : `internal/preferences` (schéma workspaces, cascade, validation), `internal/agents` (catalogue de modèles, niveaux d'effort, flags modèle/effort dans `BuildSubprocessArgs`), `internal/session` (`StartSubprocess` reçoit modèle et effort résolus), `internal/pool` (rôles implementer et fixer, config persistée), `internal/api/handlers` (`preferences`, `pool`, `ff`, `explore`, `docs`, nouveau endpoint de catalogue de modèles).
- **Frontend** : `SettingsPage` refondu en sous-onglets, sous-onglets Colonnes et Agent Pool dans `ConfigurationPage`, `AgentPoolModal` pré-remplie, hooks de préférences, locales `en`/`fr`.
- **Données** : nouveau contenu dans `preferences.json` (rétrocompatible : l'absence des champs équivaut au comportement actuel).
- **Coordination** : le change en cours `explore-session-resume-and-restart` touche `preferences.go` et `subprocess.go`, il garde sa configuration actuelle ; ce change s'implémente après lui.
