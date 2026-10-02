# Design

## Context

Motivation et périmètre : voir `proposal.md`. Contraintes observées dans le code, qui façonnent l'approche :

- **Les bornes de phase de la méthode A ne sont pas toutes persistées.** Le ff de promotion (`explore.go`, `runPromoteFF`) jette sa sortie et n'écrit aucune entrée `activity.jsonl` ni run `ff`. Le lancement (`Launch`, Ready → To Do) ne fait qu'écrire `launched: true` dans `.openspec.yaml`. Seuls les événements SSE (`ff_started`, `ff_done`, `change_updated`, `pool_updated`) marquent ces instants, et ils sont éphémères.
- **Ce qui est persisté** : les logs de conversation (chaque ligne porte un `ts` RFC3339Nano), les markers `pool_run_start` / `pool_run_end` (avec `outcome`) du run `pool`, les entrées d'activité de transition de worker et de commit git, et l'historique git. Les logs d'explore sont stockés sous `_explore/<ghostId>/chat/` puis copiés dans `<change>/chat/` à la promotion.
- **Rétention** : `changeLogRetentionDays` et `exploreLogRetentionDays` valent 15 jours par défaut.
- **Isolation possible** : le serveur lit sa configuration via `CONFIG_PATH`, `PORT` et `HOST`, et les données d'activité et de conversation vivent à côté de ce fichier de configuration. Une instance par run isole donc les données.
- **Pool** : en `full-autonomy`, le worker fusionne dans la branche de départ et exige une commande de validation explicite pour un projet Node (non concerné par le cas d'usage Go). Le change doit être commité avant d'être lancé.
- Le projet est en Go 1.25 ; les spécifications sont en français, le code en anglais.

## Goals / Non-Goals

**Goals:**
- Mesurer, de façon reproductible et sans modifier la plateforme, le temps humain / machine / total de la méthode A et le temps total de la méthode B sur le même cas d'usage.
- Rendre la source des métriques remplaçable par un futur module de métriques backend sans toucher au calcul ni au rapport.
- Produire des résultats vérifiables même après la purge des logs.

**Non-Goals:**
- Mesurer la qualité au-delà du seuil de validité, le coût ou les tokens.
- Instrumenter le backend (événements de phase persistés) : c'est l'évolution visée par l'interface de source, pas ce change.
- Piloter l'interface graphique de façon automatisée.
- Fournir une page de rapport dans l'application.

## Decisions

### D1. Un CLI Go unique : `backend/cmd/benchmark`, logique dans `backend/internal/benchmark`
Sous-commandes : `prepare` (clone + config de run), `observe` (observateur de la méthode A), `baseline` (exécute la méthode B), `collect` (copie les données brutes, évalue l'acceptation, écrit `result.json`), `report` (agrège et génère le Markdown). Le CLI n'importe que des packages en lecture (formats de logs), jamais le manager de session ni le pool.
*Alternatives* : scripts shell + `jq` (rapides mais fragiles pour médianes, YAML et tests unitaires) ; Python (introduit une seconde toolchain dans un dépôt Go + React). Go permet de tester le calcul des chronos avec `go test` comme le reste du projet.

### D2. Observateur externe plutôt que modification du backend
`observe` s'abonne au flux SSE `GET /api/workspaces/{id}/events` de l'instance du run et, à chaque `change_updated` du change suivi, interroge `GET /api/workspaces/{id}/changes/{name}` pour dériver la colonne Kanban. Il écrit `events.jsonl` (horodatage de réception + événement) dans le dossier du run. Le moment `premier message d'explore` vient du premier `ts` des logs de chat de l'exploration.
*Alternatives* : (a) déduire les bornes des mtimes de fichiers : `.openspec.yaml` est réécrit au lancement et écrase la fin du ff, donc non fiable ; (b) ajouter des entrées d'activité de phase dans le backend : plus propre (c'est l'évolution visée) mais change le produit mesuré et sort du périmètre. Précision attendue : de l'ordre de la seconde, très inférieure aux durées mesurées (minutes).

### D3. Interface de source de métriques
```
 +----------------+      +-------------------+      +-----------+
 | MetricsSource  | ---> | PhaseCalculator   | ---> | Reporter  |
 | (interface)    |      | (pur, testable)   |      | (Markdown)|
 +----------------+      +-------------------+      +-----------+
   ^           ^
   |           |
 ObserverSource  (futur) BackendMetricsSource
 (events.jsonl + logs    (lit des événements de phase
  + git)                  persistés par le backend)
```
L'interface expose des **événements de phase normalisés** (`explore_started`, `ff_started`, `ff_done`, `launched`, `pool_started`, `pool_ended(outcome)`, `merged`) avec leur instant. Le calcul et le rapport ne voient que ces événements. Une implémentation de test en mémoire sert aux tests unitaires.

### D4. Un run = un répertoire de sortie autonome
`benchmark/results/<run-id>/` contient : `config.effective.yaml`, `events.jsonl`, copie des logs d'activité et de conversation, extrait `git log`, sortie du test d'acceptation et `result.json`. `report` ne lit que les `result.json`. La rétention de la plateforme n'affecte donc plus le rapport. Une instance dédiée par run (`CONFIG_PATH` et `PORT` propres) évite de mélanger les données et garantit que tout ce qui se trouve dans le répertoire de configuration appartient au run.

### D5. Test d'acceptation caché, boîte noire, copié après le run
`benchmark/acceptance/` contient un test Go qui démarre le serveur du clone sur un workspace de fixture (un projet OpenSpec minimal aux tâches et aux runs connus), appelle `GET .../changes/{name}/stats` et vérifie le JSON. Le test est copié dans le clone seulement après la fin du run, puis exécuté. Le brief fixe le chemin et les noms de champs de la réponse mais laisse volontairement ambigu ce que compte « un run » ; la réponse scriptée de l'explore lève l'ambiguïté (et correspond à ce que le test attend).
*Pourquoi boîte noire HTTP* : les deux méthodes peuvent organiser le code différemment (noms de fonctions, packages) ; seul le contrat HTTP est comparable.

### D6. Méthode A pilotée par un opérateur avec runbook ; méthode B entièrement automatisée
Pour la méthode A, l'opérateur suit un runbook : démarrer l'instance (`prepare`), lancer `observe`, ouvrir l'UI, créer l'exploration, coller les réponses scriptées, promouvoir, commiter le change (prérequis du pool), lancer le change, laisser le pool terminer, puis `collect`. Le chrono de la phase humaine est donc une borne basse (réponses prêtes à coller) mais reste exécuté par un humain réel.
*Alternative* : piloter la WebSocket d'exploration et les endpoints par script. Reproductible à 100 % mais coûteux à écrire et à maintenir, et il masquerait la latence humaine d'orchestration (promotion, commit, lancement) que le benchmark veut justement mesurer. À reconsidérer si le benchmark est répété souvent.

### D7. Baseline : `claude -p`, une règle de relance
`baseline` exécute `claude -p` dans le clone avec le brief et les clarifications (voir la spec `benchmark-protocol`), avec le même agent, modèle et effort. Après chaque exécution, il lance la commande de validation (`go test ./...` à la racine du backend du clone). En cas d'échec, relance unique type (« les tests suivants échouent, corrige : <sortie> ») jusqu'à `max_retries` (défaut : 2, configurable). Les temps agent et validation sont mesurés séparément.
*Alternative* : session interactive libre. Plus réaliste mais non reproductible.

### D8. Validité et agrégats
Un run est valide si et seulement si : le test d'acceptation passe, l'observation est complète (méthode A) et, pour A, le run de pool a l'outcome `completed`. Les agrégats (médiane, min, max, n valides / n total) ne portent que sur les runs valides. La médiane est choisie plutôt que la moyenne à cause de la forte variance d'un agent non déterministe sur 3 runs.

### D9. Configuration
`benchmark/benchmark.yaml` (modèle : `runs: 3`, `start_sha`, `agent`, `model`, `effort`, `max_retries`, chemins du brief, des réponses, du test d'acceptation, `output_dir`). Chargé via `yaml.v3`, déjà présent dans le module. Validation stricte avant tout démarrage.

## Risks / Trade-offs

- **[Biais de familiarité]** L'agent connaît ce dépôt et la plateforme a été en partie construite avec elle-même → noté dans la section « Limites » du rapport, non corrigé.
- **[Faible échantillon]** 3 runs par méthode ne donnent qu'un ordre de grandeur → le rapport affiche min/max et n, et le nombre de runs est configurable.
- **[Observateur coupé en cours de run]** Des événements perdus fausseraient les bornes → run marqué invalide plutôt que corrigé après coup.
- **[Latence humaine variable]** Même avec des réponses scriptées, la latence de clic de l'opérateur bruite les temps humains → décomposition explicite (explore, attente de lancement) pour la lire séparément du temps machine.
- **[Test d'acceptation trop proche de l'une des méthodes]** Si le contrat est mal spécifié, une méthode peut échouer pour une raison d'interprétation → le contrat (chemin, champs, sémantique) est figé dans le brief et les réponses avant le premier run, et testé contre une implémentation de référence.
- **[Dérive de la plateforme]** Si les événements SSE ou les formats de logs changent, l'observateur casse → les tests de `ObserverSource` s'appuient sur des fixtures du format actuel, et le point de bascule vers un module backend est l'interface D3.
- **[Données personnelles dans les copies de logs]** Les logs de conversation sont copiés dans `benchmark/results/` → ajouter ce dossier au `.gitignore` et ne versionner que les rapports.

## Open Questions

- Emplacement exact des données (activité, conversations) relativement à `CONFIG_PATH` pour l'instance dédiée : à confirmer pendant l'implémentation de `prepare`, sans impact sur les specs.
- Agent et modèle exacts de la configuration par défaut : à fixer dans `benchmark.yaml` au premier run, à partir des réglages de rôle `implementer` du workspace.
