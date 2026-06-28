### Requirement: Afficher l'ExplorePanel en bottom panel
Lorsqu'une session d'exploration est active, le chat SHALL s'afficher dans un panneau ancré en bas de la page, sous les colonnes Kanban. Les colonnes Kanban SHALL occuper toute la largeur disponible indépendamment de l'état du bottom panel.

#### Scenario: Ouverture du bottom panel
- **WHEN** l'utilisateur clique sur une carte en colonne To Explore
- **THEN** un panneau de chat s'ouvre en bas de l'écran, sous les colonnes Kanban, sans modifier la largeur des colonnes

#### Scenario: Colonnes Kanban non affectées
- **WHEN** le bottom panel est ouvert
- **THEN** les colonnes Kanban conservent leur largeur totale disponible (pas de compression horizontale)

#### Scenario: Fermeture du bottom panel
- **WHEN** l'utilisateur clique sur le bouton de fermeture du bottom panel
- **THEN** le panneau disparaît et les colonnes reprennent toute la hauteur disponible

### Requirement: Redimensionnement vertical du bottom panel
L'utilisateur SHALL pouvoir ajuster la hauteur du bottom panel via un drag handle positionné sur son bord supérieur. La hauteur SHALL être contrainte entre 200px (minimum) et 70% de la hauteur de la fenêtre (maximum).

#### Scenario: Drag vers le haut agrandit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le haut
- **THEN** la hauteur du panel augmente au fur et à mesure du déplacement, dans la limite du maximum

#### Scenario: Drag vers le bas rétrécit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le bas
- **THEN** la hauteur du panel diminue au fur et à mesure du déplacement, dans la limite du minimum

#### Scenario: Relâchement de la souris fixe la hauteur
- **WHEN** l'utilisateur relâche le bouton de la souris pendant un drag
- **THEN** la hauteur du panel est fixée à la valeur courante et le drag s'arrête

### Requirement: Auto-invocation du skill explore au démarrage de session
Au démarrage d'une nouvelle session d'exploration, le backend SHALL automatiquement injecter `/opsx:explore <changeName>` comme premier message vers le subprocess Claude. Ce message d'initialisation ne SHALL PAS apparaître dans le fil de chat comme message utilisateur.

#### Scenario: Démarrage d'une nouvelle session
- **WHEN** le backend démarre un nouveau subprocess pour un changement
- **THEN** il envoie immédiatement `/opsx:explore <changeName>` sur stdin du subprocess avant tout message utilisateur

#### Scenario: Réponse initiale de Claude visible
- **WHEN** Claude répond au message d'initialisation
- **THEN** sa réponse apparaît dans le fil de chat comme premier message assistant

### Requirement: Conservation de l'historique et replay sur reconnexion
Le backend SHALL conserver en mémoire tous les messages produits par le subprocess pour la durée de vie de la session. À chaque nouvelle connexion WebSocket pour une session existante, le backend SHALL rejouer l'historique complet avant de reprendre le stream live.

#### Scenario: Reconnexion WebSocket avec historique
- **WHEN** la WebSocket est reconnectée pour une session déjà active
- **THEN** le frontend reçoit d'abord tous les messages précédents dans l'ordre, puis les nouveaux messages en temps réel

#### Scenario: Buffer limité à 500 messages
- **WHEN** la session produit plus de 500 messages
- **THEN** les messages les plus anciens sont supprimés du buffer (sliding window), les plus récents sont conservés
