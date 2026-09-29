# Spec Delta

## ADDED Requirements

### Requirement: Génération indépendante de la taille des specs
Le run de génération SHALL pouvoir être lancé et aboutir quel que soit le volume cumulé des fichiers `openspec/specs/**/spec.md` du workspace. Le contenu de ces specs SHALL NOT être transmis à l'agent via la ligne de commande de son processus ; l'agent SHALL les lire depuis le répertoire du workspace.

#### Scenario: Workspace avec un grand nombre de specs
- **WHEN** une génération est demandée pour un workspace dont les fichiers `spec.md` totalisent plusieurs centaines de kilooctets
- **THEN** le run d'agent démarre sans erreur et les pages sont générées à partir de l'ensemble des specs

#### Scenario: Workspace sans spec
- **WHEN** une génération est demandée pour un workspace sans fichier `spec.md` sous `openspec/specs/`
- **THEN** le run d'agent démarre sans erreur de lancement
