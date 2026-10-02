# Proposal

## Why

Aucune mesure ne permet aujourd'hui de savoir si développer une fonctionnalité de bout en bout avec la plateforme (colonnes `to explore` → `done` : explore, ff, pool d'agents, validation, merge) est plus rapide qu'un développement équivalent confié directement à un agent CLI sans la plateforme. Sans chiffres comparables, l'intérêt du process (specs, worktrees, boucle d'auto-guérison) reste une impression.

## What Changes

- Ajout d'un **benchmark reproductible** comparant deux méthodes sur un même cas d'usage :
  - **Méthode A (plateforme)** : explore → ff → pool (`full-autonomy`) → merge.
  - **Méthode B (baseline)** : `claude -p` direct avec le même brief, même modèle et même effort que le rôle `implementer`, avec une seule règle de relance (coller la sortie des tests en échec).
- Axe de performance retenu : **la vitesse**. Pour la méthode A, deux chronos séparés (temps humain d'explore, temps machine ff + pool + merge) plus le total. Le chrono d'un run n'est **valide que si le test d'acceptation caché passe**.
- Cas d'usage : un clone de ce dépôt à un SHA figé, avec pour fonctionnalité un endpoint Go `GET /api/workspaces/{id}/changes/{name}/stats` (avancement des tâches, nombre de runs, durée cumulée). Un test d'acceptation boîte noire, écrit avant et copié dans le clone après la fin du run, sert de juge commun.
- Un **outil d'extraction et de rapport** (CLI Go) avec un **observateur externe** du flux d'événements de la plateforme (les bornes ff et lancement ne sont pas persistées aujourd'hui), qui lit aussi les données existantes (`activity.jsonl`, runs de conversation, `git log`) derrière une interface de source de métriques, pour pouvoir basculer plus tard sur un module de métriques dans le backend sans réécrire le calcul ni le rapport.
- Un **fichier de configuration du benchmark** (nombre de runs par méthode, 3 par défaut, SHA de départ, agent/modèle/effort, chemin du brief et du test d'acceptation).
- **Export des données brutes de chaque run** avant la rétention des logs (15 jours par défaut), pour que le rapport reste vérifiable.
- **Rapport Markdown** généré : tableau par run, médiane et min/max par méthode, section « Limites » (biais de familiarité : l'agent connaît ce dépôt et la plateforme a été en partie construite avec elle-même).

Hors périmètre : qualité fonctionnelle au-delà du seuil de validité, coût et tokens (la plateforme ne capture pas l'usage aujourd'hui), page de rapport dans l'application, modification du comportement de la plateforme.

## Capabilities

### New Capabilities
- `benchmark-protocol`: configuration du benchmark, préparation d'un run reproductible (clone à SHA figé, brief, réponses d'explore scriptées, test d'acceptation caché), règle de la baseline et critère de validité d'un run.
- `benchmark-metrics-report`: extraction des chronos par phase via une interface de source de métriques, export des données brutes avant rétention, agrégation par méthode et génération du rapport Markdown.

### Modified Capabilities
<!-- Aucune : le benchmark lit les données existantes sans modifier le comportement de la plateforme. -->

## Impact

- **Code** : nouveau package `backend/internal/benchmark` et nouvelle commande `backend/cmd/benchmark` (Go 1.25, dépendances existantes : `yaml.v3`). Aucun changement dans `pool`, `activity`, `conversation` ni `api`.
- **Fichiers** : répertoire `benchmark/` à la racine pour la configuration, le brief, les réponses d'explore scriptées, le test d'acceptation et les rapports générés. Cible `make` optionnelle.
- **Données lues** : flux SSE `GET /api/workspaces/{id}/events` et `GET .../changes/{name}` de l'instance du run, `<config-dir>/activity/<workspaceID>/<change>/activity.jsonl`, logs de conversation du change (`chat`, `pool`, avec markers `pool_run_start` / `pool_run_end`), `git log` du clone.
- **Pas de modification de la plateforme** : une instance dédiée par run (`CONFIG_PATH`, `PORT` propres) ; le backend n'est ni instrumenté ni modifié.
