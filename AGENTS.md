# AGENTS.md

Spécifications en francais.
Source code en anglais.

## Objectif

Application qui permet de visualiser l'avancées des changements réalisés via OpenSpec.

## Fonctionnalités

- Kanban Board : to explore, todo, in progress, done
- Specs : pour gérer et suivre les spécifications actées

## Détails

### Kanban Board

Colonnes : 
- To Explore : pour les spécifications à explorer
  - Permer de créer une conversation avec la skill `openspec-explore` pour explorer la spécification
  - Attend en entrée de la tache
    - Description
    - Questions ouvertes
  - Attend en sortie de la tache : dans le fil de discussion de la tache
    - Réponses aux questions ouvertes
    - Résumé de la spécification explorée
  - Call to action
    - Bouton pour créer une nouvelle tache dans la colonne `To Do` via la skill `openspec-ff`
- To Do : pour les spécifications à faire
  - Suivi de l'avancement de la spécification
- In Progress : pour les spécifications en cours de réalisation
  - Suivi de l'avancement de la spécification
- Done : pour les spécifications terminées
  - Call to action
    - Bouton pour archiver la spécification via la skill `openspec-archive`

### Specs

- Liste des spécifications actées

### Workspace

Gestion des workspaces pour gérer plusieurs projets avec des Kanban et Specs dédiés.

### Configuration

Fichier de configuration `config.yaml` à la racine du projet.

- Ajouter un projet via son full path (gestion multi projets avec Kanban dédiés)
  - Fichier de configuration
  - Bouton pour ajouter un projet avec un explorateur de fichiers pour sélectionner le répertoire du projet

#### Configuration de l'Agent Gemini

L'agent Gemini nécessite d'avoir accès aux variables d'environnement Google Cloud, en particulier `GOOGLE_CLOUD_PROJECT`.
Comme le backend Go propage de manière transparente l'environnement du processus parent à ses sous-processus d'agent (le paramètre `cmd.Env` est conservé par défaut), il vous suffit de définir la variable `GOOGLE_CLOUD_PROJECT` dans l'environnement de lancement d'opensp8c :

```bash
export GOOGLE_CLOUD_PROJECT="votre-projet-gcp"
```

## Technologies

Frontend : ReactJS v19
Backend : Golang v1.25
