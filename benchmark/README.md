# Benchmark plateforme vs baseline

Mesure la **vitesse** de deux façons de développer la même fonctionnalité à partir d'un même SHA :

- **Méthode A (plateforme)** : explore → ff → pool (`full-autonomy`) → merge, dans une instance dédiée de la plateforme.
- **Méthode B (baseline)** : `claude -p` avec le même brief, les mêmes clarifications, le même modèle et le même effort, plus une règle de relance (coller la sortie des tests en échec).

Cas d'usage : l'endpoint `GET /api/workspaces/{id}/changes/{name}/stats` décrit dans `brief.md`. Un run n'est **valide** que si le test d'acceptation caché (`acceptance/`) passe, copié dans le clone *après* le run.

## Fichiers

| Fichier | Rôle |
|---|---|
| `benchmark.yaml` | Configuration : `runs` (3 par défaut), `start_sha`, `agent`, `model`, `effort`, `max_retries` (2 par défaut), chemins. |
| `brief.md` | Brief commun aux deux méthodes (ambiguïté volontaire : « run » et « durée »). |
| `answers.md` | Réponses d'explore scriptées : données à l'opérateur (A), ajoutées au prompt (B). |
| `acceptance/` | Test d'acceptation boîte noire. Jamais présent dans le clone pendant le run. |
| `results/` | Sorties par run (ignoré par git : contient des copies de logs de conversation). |

`start_sha` doit **précéder** l'ajout du dossier `benchmark/` : `prepare` refuse un clone qui le contient.

## Changer le nombre de runs

Éditer `runs:` dans `benchmark.yaml` (3 par défaut, au minimum 1). Il s'applique aux deux méthodes. `prepare` sans `-run` prépare tous les runs.

## Commandes

Depuis `backend/` (les chemins sont relatifs à la racine du dépôt, `-repo ..`) :

```bash
cd backend
go run ./cmd/benchmark prepare  -repo .. [-method A|B] [-run N]
go run ./cmd/benchmark observe  -repo .. -run a-01 -change change-stats-endpoint
go run ./cmd/benchmark baseline -repo .. -run b-01
go run ./cmd/benchmark collect  -repo .. -run <id>
go run ./cmd/benchmark report   -repo ..
```

## Runbook méthode A (plateforme)

Pour chaque run `a-NN` :

1. **Préparer** : `prepare -repo .. -method A -run N`. Affiche le clone, le répertoire de configuration du run (données d'activité et de conversation isolées), le port, l'identifiant du workspace et la commande de démarrage.
2. **Démarrer l'instance** du run avec la commande affichée (`CONFIG_PATH=… PORT=… go run ./cmd/server` depuis le clone). Les données du run vivent à côté de son `config.yaml` : `activity/<workspaceID>/<change>/activity.jsonl` et `conversations/<workspaceID>/<change>/{chat,pool,…}/`.
3. **Lancer l'observateur** avant tout message d'explore, dans un autre terminal : `observe -repo .. -run a-NN -change change-stats-endpoint`. Il écrit `results/a-NN/events.jsonl`. Un observateur qui n'a pas couvert tout le run (démarré trop tard, arrêté avant le merge, connexion perdue) rend le run invalide.
4. **Régler le pool** du workspace en `full-autonomy` avec le même agent, modèle et effort que dans `benchmark.yaml`. Valider qu'aucune commande de validation spécifique n'est requise (projet Go).
5. **Explorer** : ouvrir l'UI du run (`http://localhost:<port>`), créer une exploration, coller `brief.md`, puis répondre aux questions avec les réponses de `answers.md`, telles quelles. Nommer le change `change-stats-endpoint`.
6. **Promouvoir** l'exploration (déclenche le ff : `ff_started` → `ff_done`).
7. **Commiter le change** dans le clone (prérequis du pool) :
   `git -C <clone> add openspec && git -C <clone> commit -m "change-stats-endpoint"` (`<clone>` : chemin affiché par `prepare`, `benchmark/results/a-NN/clone`).
8. **Lancer** le change (Ready → To Do) et démarrer le pool. Ne pas intervenir ensuite.
9. Attendre le **merge** dans la branche de départ (`benchmark-start`) et l'arrêt du pool, puis arrêter l'observateur (Ctrl-C) et l'instance.
10. **Collecter** : `collect -repo .. -run a-NN` copie les données brutes dans `results/a-NN/raw/`, copie et exécute le test d'acceptation, calcule les chronos et écrit `result.json`.

Chronos de la méthode A : temps humain = explore + attente de lancement ; temps machine = ff + pool (de `todo` au commit de merge) ; total = humain + machine. Un run de pool dont l'issue n'est pas `completed` (par exemple `paused`) rend le run invalide.

## Runbook méthode B (baseline)

Pour chaque run `b-NN` :

1. `prepare -repo .. -method B -run N`.
2. `baseline -repo .. -run b-NN` : exécute `claude -p --dangerously-skip-permissions [--model …] [--effort …]` dans le clone avec le brief et les clarifications, lance `go test ./...` dans `backend/`, relance l'agent avec la sortie en échec jusqu'à `max_retries` (statut `validation_failed` ensuite). Le clone est isolé, d'où l'option de permissions.
3. `collect -repo .. -run b-NN`.

## Rapport

`report -repo ..` lit les `results/*/result.json` et écrit `results/report.md` (option `-out` pour un autre chemin, à versionner si besoin) : tableau par run, médiane/min/max sur les runs valides (« n/N valides »), écart des médianes, configuration, SHA et « Limites ». Le rapport est regénérable à partir des seuls `result.json`, même après la purge des logs par la rétention de la plateforme (15 jours par défaut).

## Limites connues

- Biais de familiarité : l'agent connaît ce dépôt, et la plateforme a été en partie construite avec elle-même.
- 3 runs donnent un ordre de grandeur, pas une preuve statistique.
- Ni la qualité au-delà du test d'acceptation, ni le coût, ni les tokens ne sont mesurés.
- Le temps humain de A est une borne basse (réponses prêtes à coller).
- Évolution prévue : un module de métriques dans le backend remplacerait `ObserverSource` derrière l'interface `MetricsSource`, sans toucher au calcul ni au rapport.
