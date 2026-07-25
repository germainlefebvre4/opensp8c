### Requirement: Modal de configuration et de lancement du pool
L'interface utilisateur SHALL inclure un bouton de lancement d'action globale dans le Kanban qui ouvre une modale de configuration du Pool d'Agents. Cette modale SHALL permettre à l'utilisateur d'ajuster le nombre de workers parallèles (de 1 à 5) et de sélectionner le Mode de Délégation global (`full-autonomy` ou `hitl-review`).

#### Scenario: Ouverture et configuration de la modale
- **WHEN** l'utilisateur clique sur le bouton "Lancer le Pool"
- **THEN** une modale s'affiche avec un sélecteur numérique pour la taille du pool, un commutateur pour le mode de délégation, et un bouton de confirmation "Démarrer"

#### Scenario: Validation et lancement du pool
- **WHEN** l'utilisateur clique sur "Démarrer" dans la modale de configuration
- **THEN** l'application envoie les paramètres au backend via l'API, ferme la modale, et affiche un indicateur visuel de pool actif dans l'en-tête du Kanban

### Requirement: Panneau interactif de Review HITL (Human-In-The-Loop)
Lorsque le mode de délégation est configuré sur `hitl-review` et qu'un changement terminé arrive dans la colonne **To Review**, l'utilisateur SHALL pouvoir cliquer sur la carte correspondante pour ouvrir un panneau de review interactif. Ce panneau SHALL afficher la liste des fichiers modifiés, un diff de code interactif, un champ de saisie de feedback textuel, un bouton "Approuver et Fusionner" et un bouton "Demander des Corrections".

#### Scenario: Affichage du diff et des fichiers modifiés
- **WHEN** l'utilisateur ouvre le panneau de review pour un changement situé dans la colonne "To Review"
- **THEN** le panneau affiche les fichiers affectés et permet de déplier un composant de rendu Diff affichant les lignes ajoutées et supprimées dans la branche isolée du worktree

#### Scenario: Approbation et fusion finale du code
- **WHEN** l'utilisateur clique sur "Approuver et Fusionner" dans le panneau de review
- **THEN** l'application envoie une requête de fusion au backend, qui fusionne la branche de feature dans la branche courante, nettoie le worktree, déplace la carte Kanban dans la colonne "Done" et ferme le panneau

#### Scenario: Demande de corrections avec feedback textuel
- **WHEN** l'utilisateur écrit un retour dans le champ de feedback et clique sur "Demander des Corrections"
- **THEN** l'application envoie le feedback au backend, qui repasse la carte du changement en colonne "In Progress", relance le worker associé avec le feedback injecté dans son invite système, et ferme le panneau de review
