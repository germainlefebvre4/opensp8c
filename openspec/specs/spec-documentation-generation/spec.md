# spec-documentation-generation Specification

## Purpose

Génère, à la demande et en un seul run d'agent, un ensemble fixe de pages de documentation lisible à partir des spécifications brutes d'un workspace, et les persiste dans un répertoire dédié et invariable du projet cible.

## Requirements

### Requirement: Déclenchement d'un run de génération unique
Le système SHALL exposer un moyen de déclencher, pour un workspace donné, un run d'agent unique chargé de produire l'ensemble des pages de documentation en une seule passe, en réutilisant l'agent par défaut configuré pour ce workspace. Un déclenchement SHALL être rejeté sans lancer de second run tant qu'un run de génération est déjà en cours pour ce workspace.

#### Scenario: Déclenchement nominal
- **WHEN** une génération est demandée pour un workspace sans run en cours
- **THEN** le système lance un run d'agent unique pour ce workspace, avec l'agent par défaut configuré

#### Scenario: Déclenchement concurrent
- **WHEN** une génération est demandée pour un workspace alors qu'un run de génération est déjà en cours pour ce même workspace
- **THEN** la demande est rejetée et le run existant se poursuit sans duplication

### Requirement: Formalisme fixe et règles de skip déterministes
Le run de génération SHALL toujours produire les pages `overview.md`, `architecture.md` et `domain-model.md`. Il SHALL produire `workflows.md` uniquement lorsque les spécifications du workspace décrivent un cycle de vie ou un enchaînement multi-étapes ; dans le cas contraire, cette page SHALL être omise plutôt que générée vide ou générique.

#### Scenario: Specs décrivant un cycle de vie multi-étapes
- **WHEN** les spécifications du workspace décrivent un enchaînement multi-étapes (ex. cycle de vie d'une entité métier, pipeline de traitement)
- **THEN** la page `workflows.md` est générée et décrit cet enchaînement

#### Scenario: Specs sans cycle de vie applicatif
- **WHEN** les spécifications du workspace ne décrivent aucun enchaînement multi-étapes (ex. bibliothèque cliente sans cycle de vie utilisateur)
- **THEN** la page `workflows.md` n'est pas générée, et aucune page vide ou de substitution n'est créée à sa place

#### Scenario: Pages toujours produites
- **WHEN** un run de génération aboutit, quel que soit le contenu des spécifications
- **THEN** les pages `overview.md`, `architecture.md` et `domain-model.md` sont générées

### Requirement: Persistance fixe sous docs/opensp8c/
Les pages générées SHALL être écrites dans le répertoire `docs/opensp8c/` à la racine du projet cible, ce chemin de répertoire étant fixe et identique quel que soit le projet cible. La génération SHALL ne jamais modifier ni supprimer les fichiers de documentation déjà présents ailleurs dans le projet cible, y compris dans son propre répertoire `docs/`.

#### Scenario: Projet cible sans documentation existante
- **WHEN** un run de génération aboutit sur un projet cible sans répertoire `docs/`
- **THEN** le répertoire `docs/opensp8c/` est créé et contient les pages générées

#### Scenario: Projet cible avec documentation écrite à la main
- **WHEN** un run de génération aboutit sur un projet cible dont le répertoire `docs/` contient déjà des fichiers écrits à la main
- **THEN** ces fichiers restent inchangés et seul le sous-répertoire `docs/opensp8c/` est créé ou mis à jour

### Requirement: Diagrammes Mermaid dans le contenu généré
Lorsque les spécifications permettent d'identifier une relation ou un enchaînement représentable graphiquement (relations entre concepts du domaine, architecture des composants, étapes d'un flux), le contenu généré SHALL inclure un diagramme au format Mermaid correspondant, intégré dans la page Markdown concernée.

#### Scenario: Relations de domaine identifiables
- **WHEN** les spécifications décrivent plusieurs concepts métier liés entre eux
- **THEN** `domain-model.md` inclut un bloc de code Mermaid représentant ces relations

#### Scenario: Absence de relation représentable
- **WHEN** une page générée ne comporte aucune relation ou enchaînement suffisamment structuré pour un diagramme
- **THEN** cette page est produite en contenu texte seul, sans bloc Mermaid forcé

### Requirement: Consultation des pages générées et de leur fraîcheur
Le système SHALL exposer un moyen de lister les pages actuellement présentes sous `docs/opensp8c/` d'un workspace, d'en lire le contenu, et de déterminer si la documentation générée est potentiellement obsolète en comparant la date de modification la plus récente parmi `openspec/specs/**/spec.md` à celle des pages générées.

#### Scenario: Liste des pages existantes
- **WHEN** les pages générées sont consultées pour un workspace
- **THEN** seules les pages effectivement présentes sous `docs/opensp8c/` sont retournées

#### Scenario: Spec plus récente que la documentation générée
- **WHEN** au moins un fichier `spec.md` sous `openspec/specs/` a une date de modification postérieure à toutes les pages sous `docs/opensp8c/`
- **THEN** la documentation générée est signalée comme potentiellement obsolète

#### Scenario: Documentation générée à jour
- **WHEN** toutes les pages sous `docs/opensp8c/` sont plus récentes que tous les fichiers `spec.md` sous `openspec/specs/`
- **THEN** la documentation générée n'est pas signalée comme obsolète

### Requirement: Langue de la documentation générée
Le run de génération SHALL produire les pages de documentation dans la langue `documentation` résolue au moment du lancement, telle que définie par `agent-language-settings`. Les noms de fichiers des pages (`overview.md`, `architecture.md`, `domain-model.md`, `workflows.md`) et le chemin `docs/opensp8c/` SHALL rester inchangés quelle que soit la langue.

#### Scenario: Documentation dans la langue de l'application
- **WHEN** la langue `documentation` vaut `auto`, la langue de l'application est `fr`, et une génération aboutit
- **THEN** les pages sous `docs/opensp8c/` sont rédigées en français

#### Scenario: Documentation dans une langue explicite
- **WHEN** la langue `documentation` vaut `en` alors que la langue de l'application est `fr`, et une génération aboutit
- **THEN** les pages sous `docs/opensp8c/` sont rédigées en anglais

#### Scenario: Noms de fichiers indépendants de la langue
- **WHEN** une génération aboutit avec une langue `documentation` autre que l'anglais
- **THEN** les pages portent exactement les noms de fichiers fixes définis pour la génération

### Requirement: Génération indépendante de la taille des specs
Le run de génération SHALL pouvoir être lancé et aboutir quel que soit le volume cumulé des fichiers `openspec/specs/**/spec.md` du workspace. Le contenu de ces specs SHALL NOT être transmis à l'agent via la ligne de commande de son processus ; l'agent SHALL les lire depuis le répertoire du workspace.

#### Scenario: Workspace avec un grand nombre de specs
- **WHEN** une génération est demandée pour un workspace dont les fichiers `spec.md` totalisent plusieurs centaines de kilooctets
- **THEN** le run d'agent démarre sans erreur et les pages sont générées à partir de l'ensemble des specs

#### Scenario: Workspace sans spec
- **WHEN** une génération est demandée pour un workspace sans fichier `spec.md` sous `openspec/specs/`
- **THEN** le run d'agent démarre sans erreur de lancement
