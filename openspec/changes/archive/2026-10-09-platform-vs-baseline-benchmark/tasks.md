# Tasks

## 1. Configuration et squelette du CLI

- [x] 1.1 Créer `backend/cmd/benchmark` et `backend/internal/benchmark` avec les sous-commandes `prepare`, `observe`, `baseline`, `collect`, `report` (aide uniquement) et vérifier que `go run ./cmd/benchmark --help` liste les cinq sous-commandes
- [x] 1.2 Implémenter le chargement et la validation de `benchmark/benchmark.yaml` (`runs` à 3 par défaut, `start_sha`, `agent`, `model`, `effort`, `max_retries` à 2 par défaut, chemins du brief, des réponses, du test d'acceptation, `output_dir`) et vérifier par tests unitaires : défauts appliqués, `runs: 0` refusé, SHA inconnu refusé, fichier référencé absent refusé, chaque message nommant le champ en cause
- [x] 1.3 Ajouter `benchmark/benchmark.yaml` d'exemple et `benchmark/results/` au `.gitignore`, puis vérifier qu'un `git status` après un run factice ne montre aucun fichier de résultat

## 2. Préparation d'un run

- [x] 2.1 Implémenter `prepare` : clone isolé au SHA de départ, arbre propre, absence du test d'acceptation et du dossier `benchmark/` dans le clone ; vérifier par test sur un dépôt temporaire (HEAD égal au SHA, test d'acceptation introuvable)
- [x] 2.2 Pour la méthode A, générer un répertoire de configuration dédié au run, un `config.yaml` avec un workspace pointant sur le clone, un port propre au run et la commande de démarrage de l'instance ; vérifier par test que deux runs préparés ont des répertoires et des ports distincts
- [x] 2.3 Confirmer l'emplacement des données d'activité et de conversation relativement à `CONFIG_PATH` en démarrant une instance dédiée, et consigner le résultat dans le runbook (task 6.2)

## 3. Source de métriques et calcul des chronos

- [x] 3.1 Définir l'interface `MetricsSource` et les événements de phase normalisés (`explore_started`, `ff_started`, `ff_done`, `launched`, `pool_started`, `pool_ended`, `merged`), plus une source en mémoire pour les tests ; vérifier qu'elle compile et sert à un test de calcul
- [x] 3.2 Implémenter le calculateur de chronos de la méthode A (humain = explore + attente de lancement, machine = ff + pool, total = humain + machine, cumul de plusieurs runs de pool, invalidité sur outcome différent de `completed`) et vérifier par tests tabulaires, dont les scénarios « pool en pause » et « plusieurs runs de pool »
- [x] 3.3 Implémenter le calculateur de la méthode B (total, temps agent, temps validation, nombre de relances) et vérifier par un test avec une relance

## 4. Observateur et source par défaut

- [x] 4.1 Implémenter `observe` : abonnement au flux SSE du workspace, relevé de la colonne Kanban via l'API à chaque `change_updated`, écriture de `events.jsonl` ; vérifier avec un faux serveur HTTP/SSE que les événements et les transitions `ready` → `todo` sont enregistrés avec leur horodatage
- [x] 4.2 Gérer la perte de connexion : tentatives de reconnexion, puis marquage « observation incomplète » rendant le run invalide ; vérifier par test avec un faux serveur qui coupe la connexion
- [x] 4.3 Implémenter `ObserverSource` (événements de l'observateur + logs de chat et de pool avec leurs `ts` et markers + commit de merge via `git log`) et vérifier avec des fixtures du format de logs actuel que les événements de phase normalisés sont produits dans le bon ordre

## 5. Baseline, acceptation et collecte

- [x] 5.1 Implémenter `baseline` : exécution de `claude -p` avec le brief et les clarifications dans le clone, validation `go test ./...`, relance type avec la sortie en échec jusqu'à `max_retries`, statut d'échec de validation ensuite ; vérifier avec un faux exécutable d'agent les trois scénarios (0 relance, 1 relance, maximum atteint)
- [x] 5.2 Écrire le brief, les réponses d'explore scriptées et le test d'acceptation boîte noire `benchmark/acceptance/` (démarre le serveur du clone sur un workspace de fixture, appelle `GET .../changes/{name}/stats`) ; vérifier qu'il échoue sur le SHA de départ et passe sur une implémentation de référence
- [x] 5.3 Implémenter `collect` : copie des données brutes (journal de l'observateur, activité, conversations, extrait git) avant tout calcul, copie puis exécution du test d'acceptation, calcul des chronos et écriture de `result.json` autonome ; vérifier par test qu'un run dont l'acceptation échoue est conservé avec son statut invalide, et que `result.json` suffit à relire le run sans les logs d'origine

## 6. Rapport et documentation

- [x] 6.1 Implémenter `report` : tableau par run, agrégats (médiane, min, max, « n/N valides ») sur les seuls runs valides, écart des médianes, configuration effective, SHA, section « Limites » (biais de familiarité, taille d'échantillon, exclusion de la qualité, du coût et des tokens) ; vérifier avec des tests de type golden file, dont « 2/3 valides » et « aucune donnée valide »
- [x] 6.2 Rédiger `benchmark/README.md` avec le runbook de la méthode A (démarrer l'instance, lancer `observe`, explorer avec les réponses scriptées, promouvoir, commiter le change, lancer, `collect`), le déroulé de la méthode B et la façon de changer `runs` ; vérifier en suivant le document tel qu'écrit sur une préparation à blanc
- [x] 6.3 Ajouter un test garantissant que le calcul et le rapport n'importent aucun package de lecture de fichiers de la plateforme (seule `ObserverSource` le fait), pour protéger le point de bascule vers un module backend

## 7. Intégration

- [x] 7.1 Exécuter un run complet de chaque méthode sur le cas d'usage, produire un rapport avec `report` et vérifier que les deux `result.json`, le journal de l'observateur et le rapport Markdown existent
- [x] 7.2 Lancer `go test ./...` dans `backend` et vérifier qu'aucun test existant n'est affecté par les nouveaux packages
