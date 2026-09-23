### Requirement: Modal de configuration et de lancement du pool
L'interface utilisateur SHALL inclure un bouton de lancement d'action globale dans le Kanban. Lorsqu'aucun pool n'est actif, ce bouton SHALL ouvrir une modale de configuration du Pool d'Agents permettant à l'utilisateur d'ajuster le nombre de workers parallèles (de 1 à 5) et de sélectionner le Mode de Délégation global (`full-autonomy` ou `hitl-review`). Lorsqu'un pool est déjà actif, ce même bouton SHALL ouvrir le panneau d'état du pool (voir "Panneau d'état du pool actif") au lieu de la modale de configuration.

#### Scenario: Ouverture et configuration de la modale (aucun pool actif)
- **WHEN** l'utilisateur clique sur le bouton "Lancer le Pool" et qu'aucun pool n'est en cours d'exécution
- **THEN** une modale s'affiche avec un sélecteur numérique pour la taille du pool, un commutateur pour le mode de délégation, et un bouton de confirmation "Démarrer"

#### Scenario: Validation et lancement du pool
- **WHEN** l'utilisateur clique sur "Démarrer" dans la modale de configuration
- **THEN** l'application envoie les paramètres au backend via l'API, ferme la modale, et affiche un indicateur visuel de pool actif dans l'en-tête du Kanban

#### Scenario: Clic sur le bouton pendant qu'un pool tourne déjà
- **WHEN** l'utilisateur clique sur le bouton d'en-tête alors qu'un pool est déjà en cours d'exécution
- **THEN** le panneau d'état du pool s'ouvre à la place de la modale de configuration ; la modale de configuration ne s'affiche pas

### Requirement: Synchronisation de l'état du bouton avec l'état réel du backend
L'indicateur "pool actif/inactif" affiché sur le bouton d'en-tête du Kanban SHALL refléter l'état réel du backend (`GET /workspaces/{id}/pool/status`) et non un état local purement optimiste. Cet état SHALL être récupéré au chargement du Kanban et SHALL rester à jour tant que la page reste ouverte, sans que l'utilisateur ait besoin de recharger la page.

#### Scenario: Rechargement de la page pendant qu'un pool tourne
- **WHEN** un pool est en cours d'exécution côté backend et que l'utilisateur recharge la page du Kanban
- **THEN** le bouton d'en-tête affiche immédiatement l'état "pool actif" (et non l'état par défaut "Lancer le Pool"), sans action supplémentaire de l'utilisateur

#### Scenario: Arrêt du pool déclenché depuis une autre session
- **WHEN** le pool est arrêté (par exemple depuis une autre fenêtre ou session ouverte sur le même workspace) pendant que l'utilisateur consulte le Kanban
- **THEN** le bouton d'en-tête et le panneau d'état, si ouvert, reflètent l'arrêt du pool sans que l'utilisateur ait à rafraîchir manuellement la page

### Requirement: Panneau d'état du pool actif
Lorsqu'un pool est en cours d'exécution, l'utilisateur SHALL pouvoir consulter un panneau listant chaque worker actif avec le change auquel il est assigné et son statut (`idle`, `working`, `testing`, `healing`, ou `paused`). Ce panneau SHALL exposer une action "Stop Pool" qui arrête l'ensemble du pool. Ce panneau ne fournit pas d'action d'arrêt individuelle par worker.

#### Scenario: Affichage de la liste des workers actifs
- **WHEN** l'utilisateur ouvre le panneau d'état alors que 2 workers sur 3 sont assignés à des changes
- **THEN** le panneau affiche les 2 workers actifs avec le nom de leur change assigné et leur statut courant, et indique que le 3e worker est disponible (idle ou non affiché comme assigné)

#### Scenario: Arrêt du pool depuis le panneau
- **WHEN** l'utilisateur clique sur "Stop Pool" dans le panneau d'état
- **THEN** l'application envoie la requête d'arrêt au backend, le panneau se ferme, et le bouton d'en-tête repasse à l'état "Lancer le Pool"

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
