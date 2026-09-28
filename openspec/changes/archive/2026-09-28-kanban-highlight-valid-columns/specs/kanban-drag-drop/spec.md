# Spec Delta

## MODIFIED Requirements

### Requirement: Indicateur visuel de drag en cours
Dès le début d'un drag (saisie de la carte), le Kanban SHALL afficher immédiatement un indicateur visuel léger de zone de dépôt (bordure ou fond) sur chacune des colonnes cibles autorisées pour la colonne source de la carte draguée, que le curseur survole ces colonnes ou non. Parmi les colonnes autorisées, celle actuellement survolée par le curseur (sur son espace vide ou sur une de ses cartes) SHALL afficher un indicateur visuel renforcé par rapport aux autres colonnes autorisées non survolées. Les colonnes non autorisées pour cette source ne SHALL afficher aucun indicateur de dépôt, y compris lorsqu'elles sont survolées par le curseur.

#### Scenario: Début du drag — toutes les colonnes autorisées s'allument
- **WHEN** l'utilisateur saisit une carte et commence à la déplacer
- **THEN** toutes les colonnes cibles autorisées pour la colonne source de cette carte affichent immédiatement un indicateur visuel léger, avant même que le curseur ne les survole

#### Scenario: Survol d'une colonne acceptante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui accepte ce drop (sur son espace vide ou sur une de ses cartes)
- **THEN** cette colonne affiche un indicateur visuel renforcé par rapport aux autres colonnes autorisées, qui conservent leur indicateur léger

#### Scenario: Survol d'une colonne rejetante
- **WHEN** l'utilisateur drag une carte au-dessus d'une colonne qui n'accepte pas ce drop
- **THEN** aucun indicateur, léger ou renforcé, n'est affiché sur cette colonne, même au survol

#### Scenario: Fin du drag
- **WHEN** l'utilisateur relâche la carte, que le drop soit accepté ou refusé
- **THEN** tous les indicateurs de zone de dépôt disparaissent de toutes les colonnes
