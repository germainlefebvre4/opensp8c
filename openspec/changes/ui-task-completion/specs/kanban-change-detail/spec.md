## MODIFIED Requirements

### Requirement: Afficher la liste des tâches dans le DetailPanel
Le `DetailPanel` SHALL afficher la liste complète des tâches du change avec leur état (faite / non faite), lue depuis le fichier `tasks.md` via le backend. Chaque tâche SHALL être rendue comme un checkbox interactif permettant de toggler son état directement depuis le panel.

#### Scenario: Change avec tasks.md
- **WHEN** le DetailPanel s'ouvre pour un change ayant un `tasks.md`
- **THEN** chaque tâche est affichée avec un checkbox dont l'état reflète `[ ]` ou `[x]`

#### Scenario: Change sans tasks.md
- **WHEN** le DetailPanel s'ouvre pour un change sans `tasks.md`
- **THEN** le panneau affiche un message indiquant qu'aucune tâche n'a été définie, sans erreur

#### Scenario: Clic sur un checkbox non coché
- **WHEN** l'utilisateur clique sur le checkbox d'une tâche non complétée
- **THEN** le checkbox est désactivé pendant la requête, puis passe à l'état coché une fois le serveur ayant confirmé la mise à jour

#### Scenario: Clic sur un checkbox coché
- **WHEN** l'utilisateur clique sur le checkbox d'une tâche complétée
- **THEN** le checkbox est désactivé pendant la requête, puis repasse à l'état non coché une fois le serveur ayant confirmé la mise à jour

#### Scenario: Erreur lors du toggle
- **WHEN** la requête PATCH échoue
- **THEN** le checkbox retrouve son état initial et une notification d'erreur est affichée
