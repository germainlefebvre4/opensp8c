## ADDED Requirements

### Requirement: Prise en charge résiliente de l'agent Gemini
Le système SHALL démarrer l'agent Gemini en utilisant ses options natives supportées et s'assurer que ses flux d'entrée et de sortie sont correctement adaptés au format de l'application.

#### Scenario: Démarrage de l'agent Gemini sans échec
- **WHEN** l'agent par défaut est "gemini" et qu'une nouvelle session d'exploration est démarrée
- **THEN** le sous-processus gemini est lancé avec les arguments d'exécution adaptés
- **THEN** l'agent démarre correctement sans émettre d'erreur d'arguments inconnus
- **THEN** la session d'exploration s'ouvre avec succès
