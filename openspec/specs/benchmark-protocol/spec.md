# benchmark-protocol Specification

## Purpose

Définit le protocole reproductible du benchmark de vitesse comparant le développement de bout en bout avec la plateforme (méthode A) à un développement direct par un agent CLI sans la plateforme (méthode B) : configuration, préparation d'un run, règle de la baseline et critère de validité.

## Requirements

### Requirement: Configuration du benchmark
Le benchmark SHALL être piloté par un fichier de configuration unique contenant au minimum : le nombre de runs par méthode (`runs`, 3 par défaut), le SHA du dépôt de départ, l'agent, le modèle et l'effort utilisés par les deux méthodes, le chemin du brief, le chemin des réponses d'explore scriptées, le chemin du test d'acceptation et le dossier de sortie des résultats. Une valeur absente SHALL prendre sa valeur par défaut ; une valeur invalide (nombre de runs inférieur à 1, SHA introuvable dans le dépôt, fichier référencé inexistant) SHALL faire échouer la commande avec un message nommant le champ en cause, avant tout démarrage de run.

#### Scenario: Nombre de runs par défaut
- **WHEN** le fichier de configuration ne définit pas `runs`
- **THEN** le benchmark prévoit 3 runs pour la méthode A et 3 runs pour la méthode B

#### Scenario: Nombre de runs surchargé
- **WHEN** le fichier de configuration définit `runs: 5`
- **THEN** le benchmark prévoit 5 runs pour chacune des deux méthodes

#### Scenario: Configuration invalide
- **WHEN** le SHA de départ configuré n'existe pas dans le dépôt
- **THEN** la commande échoue avant de créer le moindre clone et le message d'erreur nomme le champ du SHA

### Requirement: Préparation d'un run reproductible
La préparation d'un run SHALL créer un clone du dépôt isolé, positionné sur le SHA de départ configuré, distinct de tout autre run et du dépôt de travail de l'utilisateur. Le clone SHALL NOT contenir le test d'acceptation, ni aucun fichier du dossier du benchmark, pendant le run. Pour la méthode A, la préparation SHALL fournir un répertoire de configuration de la plateforme propre au run (données d'activité et de conversation isolées) et un workspace pointant sur le clone. La plateforme du run SHALL être une instance dédiée démarrée à partir de ce répertoire de configuration, sur un port propre au run.

#### Scenario: Clone à SHA figé
- **WHEN** un run est préparé avec un SHA de départ donné
- **THEN** le HEAD du clone est ce SHA et son arbre de travail est propre

#### Scenario: Test d'acceptation absent pendant le run
- **WHEN** un run est préparé
- **THEN** le test d'acceptation n'existe nulle part dans l'arbre du clone

#### Scenario: Données de plateforme isolées
- **WHEN** deux runs de méthode A sont préparés
- **THEN** chacun dispose de son propre répertoire de configuration, de sorte que les logs d'activité et de conversation d'un run ne se mélangent jamais avec ceux d'un autre

### Requirement: Brief et réponses d'explore communs aux deux méthodes
Le benchmark SHALL utiliser le même brief, mot pour mot, pour les deux méthodes. Le brief SHALL contenir au moins une ambiguïté volontaire dont la levée est fournie par un fichier de réponses scriptées. Dans la méthode A, ces réponses SHALL être celles que l'opérateur donne telles quelles pendant la phase explore. Dans la méthode B, les mêmes réponses SHALL être ajoutées au prompt initial sous forme de clarifications, afin que les deux méthodes disposent de la même information.

#### Scenario: Information identique
- **WHEN** un run de méthode A et un run de méthode B sont comparés
- **THEN** le contenu du brief et celui des réponses d'explore transmis à l'agent sont identiques dans les deux

### Requirement: Règle de la baseline
La méthode B SHALL exécuter l'agent CLI en mode non interactif avec le brief et les clarifications, dans le clone, avec les mêmes agent, modèle et effort que ceux configurés pour la méthode A. Après l'exécution, le benchmark SHALL lancer la commande de validation du clone. Si elle échoue, le benchmark SHALL relancer l'agent avec une unique consigne type contenant la sortie de la validation, dans la limite d'un nombre maximal de relances configurable. Le nombre de relances effectuées SHALL être enregistré dans le résultat du run.

#### Scenario: Validation réussie du premier coup
- **WHEN** la validation réussit après la première exécution de l'agent
- **THEN** aucune relance n'est effectuée et le résultat enregistre 0 relance

#### Scenario: Relance sur échec de validation
- **WHEN** la validation échoue après une exécution de l'agent et que le maximum de relances n'est pas atteint
- **THEN** l'agent est relancé avec la consigne type contenant la sortie de la validation et le résultat compte une relance de plus

#### Scenario: Maximum de relances atteint
- **WHEN** la validation échoue encore alors que le maximum de relances est atteint
- **THEN** le run se termine avec le statut d'échec de validation et n'est pas valide pour le calcul des médianes

### Requirement: Critère de validité d'un run
Un run SHALL être déclaré valide uniquement si le test d'acceptation caché, copié dans le clone après la fin du run, passe. Un run dont le test d'acceptation échoue, ou n'a pas pu être exécuté, SHALL être conservé dans les résultats avec son statut et ses chronos, mais SHALL être exclu des agrégats de vitesse.

#### Scenario: Run valide
- **WHEN** le test d'acceptation passe après copie dans le clone
- **THEN** le run est marqué valide et inclus dans les agrégats

#### Scenario: Run invalide conservé
- **WHEN** le test d'acceptation échoue
- **THEN** le run est marqué invalide, ses chronos restent consultables dans le résultat brut, et il n'entre dans aucune médiane ni aucun min/max

### Requirement: Résultat brut d'un run
Chaque run SHALL produire un fichier de résultat structuré dans le dossier de sortie, contenant : la méthode, l'identifiant du run, le SHA de départ, la configuration effective (agent, modèle, effort), les instants de début et de fin de chaque phase, le statut de validité et, pour la méthode B, le nombre de relances. Ce fichier SHALL suffire à regénérer le rapport sans accès aux logs d'origine.

#### Scenario: Résultat autonome
- **WHEN** les logs d'origine d'un run ont été purgés par la rétention
- **THEN** le rapport peut toujours être regénéré à partir des fichiers de résultat bruts
