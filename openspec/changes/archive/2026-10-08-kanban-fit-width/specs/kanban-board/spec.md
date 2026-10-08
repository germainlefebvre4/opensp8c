# Spec Delta

## MODIFIED Requirements

### Requirement: Application pleine largeur avec colonnes auto-adaptées
Le Kanban Board SHALL occuper toute la largeur disponible de la zone de contenu, que le DetailPanel soit ouvert ou non. Chaque colonne active SHALL avoir une largeur minimale de **190px**, et les espacements entre colonnes et autour du conteneur SHALL être resserrés de façon à ce que les six slots dépliés tiennent dans environ 1200px. Lorsque le DetailPanel est ouvert, il SHALL partager l'espace horizontal avec les colonnes (voir `kanban-change-detail`). Le bottom panel d'exploration n'affecte pas la largeur des colonnes.

Lorsque l'espace horizontal disponible est insuffisant, le Kanban SHALL dégrader son affichage selon l'échelle suivante, en passant au palier suivant uniquement si le palier courant ne suffit pas :
1. tous les slots dépliés et DetailPanel poussant les colonnes ;
2. DetailPanel rétréci jusqu'à sa largeur minimale ;
3. slot **Done/Archived** replié en rail ;
4. DetailPanel en overlay par-dessus les colonnes ;
5. scroll horizontal des colonnes, en dernier recours.

#### Scenario: Redimensionnement de la fenêtre sans panel
- **WHEN** l'utilisateur redimensionne la fenêtre du navigateur et aucun panel n'est ouvert
- **THEN** les colonnes s'adaptent automatiquement pour remplir toute la largeur disponible sans débordement horizontal tant que la largeur disponible couvre le minimum des six slots dépliés

#### Scenario: DetailPanel ouvert — colonnes réduites
- **WHEN** le DetailPanel est ouvert
- **THEN** les colonnes occupent l'espace restant après le slot du DetailPanel, et si cet espace est insuffisant l'échelle de dégradation s'applique palier par palier, le scroll horizontal n'intervenant qu'en dernier recours

#### Scenario: DetailPanel ouvert sur un écran large
- **WHEN** le DetailPanel est ouvert et la largeur disponible permet de conserver les six slots dépliés à leur largeur minimale
- **THEN** le panel pousse les colonnes, aucun slot n'est replié et aucune scrollbar horizontale n'apparaît

#### Scenario: DetailPanel ouvert — espace insuffisant
- **WHEN** le DetailPanel est ouvert et les six slots dépliés ne tiennent plus à côté du panel, même rétréci à sa largeur minimale
- **THEN** le slot Done/Archived se replie automatiquement en rail avant tout passage en overlay ou en scroll horizontal

#### Scenario: DetailPanel fermé — colonnes pleine largeur
- **WHEN** le DetailPanel est fermé
- **THEN** les colonnes reprennent toute la largeur disponible et le slot Done/Archived se redéplie automatiquement, sauf surcharge manuelle

#### Scenario: Bottom panel ouvert — largeur colonnes inchangée
- **WHEN** le bottom panel d'exploration est ouvert
- **THEN** les colonnes conservent leur largeur (le bottom panel n'affecte que la hauteur disponible)

#### Scenario: Écran très étroit — scroll horizontal en dernier recours
- **WHEN** la largeur disponible est inférieure à celle requise par cinq slots dépliés plus le rail, même avec le DetailPanel en overlay ou fermé
- **THEN** les colonnes sont scrollables horizontalement

## ADDED Requirements

### Requirement: Slot Done/Archived repliable en rail
Le slot partagé **Done/Archived** SHALL pouvoir être affiché sous forme de **rail** : une bande étroite d'environ 40px qui conserve visible le compteur de la colonne Done et un chevron d'expansion, et qui masque les cartes de Done et d'Archived. Le repli et le dépli SHALL être décidés automatiquement selon l'espace disponible (échelle de dégradation de « Application pleine largeur avec colonnes auto-adaptées »). Le chevron du slot SHALL permettre à l'utilisateur de surcharger manuellement l'état automatique ; la surcharge manuelle SHALL être prioritaire sur le calcul automatique, SHALL être conservée en mémoire pour la durée de la session et SHALL NOT être persistée entre deux sessions. Lorsque l'utilisateur bascule le slot vers l'état que le calcul automatique aurait choisi, la surcharge SHALL être levée et le comportement automatique reprend. Le repli du slot SHALL NOT modifier l'état collapse propre à la colonne Archived.

#### Scenario: Repli automatique à l'ouverture du DetailPanel
- **WHEN** le DetailPanel s'ouvre et que l'espace disponible ne permet plus six slots dépliés, sans surcharge manuelle active
- **THEN** le slot Done/Archived se replie en rail et son compteur Done reste visible

#### Scenario: Dépli automatique à la fermeture du DetailPanel
- **WHEN** le DetailPanel se ferme et que l'espace permet de nouveau six slots dépliés, sans surcharge manuelle active
- **THEN** le slot Done/Archived se redéplie

#### Scenario: Surcharge manuelle — forcer le dépli
- **WHEN** le slot est replié automatiquement et l'utilisateur clique sur le chevron du rail
- **THEN** le slot se déplie et reste déplié tant que la surcharge est active, et si l'espace manque l'échelle de dégradation poursuit avec les paliers suivants (overlay du panel, puis scroll)

#### Scenario: Surcharge manuelle — forcer le repli
- **WHEN** le slot est déplié et l'utilisateur le replie manuellement via le chevron
- **THEN** le slot reste replié même si l'espace permettrait de le déplier

#### Scenario: Retour au comportement automatique
- **WHEN** une surcharge manuelle est active et l'utilisateur bascule le slot vers l'état que le calcul automatique choisirait
- **THEN** la surcharge est levée et le slot suit de nouveau l'espace disponible

#### Scenario: Surcharge non persistée
- **WHEN** l'utilisateur recharge l'application après avoir surchargé manuellement l'état du slot
- **THEN** le slot démarre en mode automatique
