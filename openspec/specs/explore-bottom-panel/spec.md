### Requirement: Bouton delete dans le panel d'exploration d'un ghost card
Le panel d'exploration d'un ghost card SHALL afficher un bouton delete dans son header. Ce bouton déclenche une dialog de confirmation avant suppression.

#### Scenario: Bouton delete visible dans le header du panel ghost
- **WHEN** le panel d'exploration est ouvert pour un ghost card (`is_ghost: true`)
- **THEN** le header affiche un bouton delete (icône 🗑) à côté du bouton de fermeture

#### Scenario: Bouton delete absent pour les sessions named non-ghost
- **WHEN** le panel d'exploration est ouvert pour un change normal en "to-explore" (non-ghost)
- **THEN** aucun bouton delete n'est affiché dans le header

#### Scenario: Clic sur delete — dialog de confirmation
- **WHEN** l'utilisateur clique sur le bouton delete dans le header du panel ghost
- **THEN** une dialog s'affiche avec le texte "Abandonner cette exploration ?" et "La conversation sera perdue." et deux boutons : [Annuler] et [Abandonner]

#### Scenario: Confirmation de suppression — panel fermé + ghost supprimé
- **WHEN** l'utilisateur clique sur [Abandonner] dans la dialog de confirmation
- **THEN** le panel se ferme, la session backend est stoppée, le ghost record est supprimé de `preferences.json` via `DELETE /api/workspaces/{id}/explorations/{ghostId}`, et localStorage est nettoyé

#### Scenario: Annulation de suppression — panel reste ouvert
- **WHEN** l'utilisateur clique sur [Annuler] dans la dialog de confirmation
- **THEN** la dialog se ferme et le panel d'exploration reste ouvert sans modification

### Requirement: Bouton de maximisation dans le header du panel d'exploration
Le header du panel d'exploration (qu'il soit named ou anonymous) SHALL afficher un bouton de maximisation/minimisation (icônes Maximize2 / Minimize2) à côté des boutons de mode de rendu et avant le bouton de fermeture (X).

#### Scenario: Clic sur maximiser agrandit le panel
- **WHEN** le panel est en taille normale et l'utilisateur clique sur le bouton de maximisation
- **THEN** le panel passe en mode maximisé, occupant toute la hauteur de l'espace disponible sous la barre de navigation globale, et l'icône du bouton change pour indiquer le retour à la taille normale

#### Scenario: Clic sur minimiser restaure la taille normale
- **WHEN** le panel est en mode maximisé et l'utilisateur clique sur le bouton de minimisation
- **THEN** le panel quitte le mode maximisé et reprend sa hauteur redimensionnée précédente (ou la hauteur par défaut)

#### Scenario: Le drag handle est désactivé en mode maximisé
- **WHEN** le panel est en mode maximisé
- **THEN** le drag handle du bord supérieur n'est plus draggable et l'icône de curseur de redimensionnement vertical n'est pas affichée

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
L'utilisateur SHALL pouvoir ajuster la hauteur du bottom panel via un drag handle positionné sur son bord supérieur. La hauteur SHALL être contrainte entre 200px (minimum) et 90% de la hauteur de la fenêtre (maximum).

#### Scenario: Drag vers le haut agrandit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le haut
- **THEN** la hauteur du panel augmente au fur et à mesure du déplacement, dans la limite du maximum (90% de la hauteur de la fenêtre)

#### Scenario: Drag vers le bas rétrécit le panel
- **WHEN** l'utilisateur clique sur le drag handle et déplace la souris vers le bas
- **THEN** la hauteur du panel diminue au fur et à mesure du déplacement, dans la limite du minimum (200px)

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

### Requirement: Saisie multiligne dans la conversation d'exploration
Le champ de saisie des panels d'exploration SHALL être un textarea auto-redimensionnable. L'utilisateur SHALL pouvoir insérer une nouvelle ligne avec `Shift+Enter`. La touche `Enter` seule SHALL envoyer le message sans insérer de nouvelle ligne.

#### Scenario: Shift+Enter insère une nouvelle ligne
- **WHEN** l'utilisateur appuie sur `Shift+Enter` dans le champ de saisie
- **THEN** une nouvelle ligne est insérée dans le message et le message n'est pas envoyé

#### Scenario: Enter seul envoie le message
- **WHEN** l'utilisateur appuie sur `Enter` sans maintenir `Shift`
- **THEN** le message est envoyé et le champ de saisie est vidé

#### Scenario: Auto-resize du textarea
- **WHEN** l'utilisateur saisit du texte sur plusieurs lignes
- **THEN** le textarea augmente en hauteur pour afficher tout le contenu sans scrollbar interne, dans la limite d'une hauteur maximale

#### Scenario: Reset de la hauteur après envoi
- **WHEN** le message est envoyé et le champ de saisie est vidé
- **THEN** le textarea reprend sa hauteur initiale (une ligne)
