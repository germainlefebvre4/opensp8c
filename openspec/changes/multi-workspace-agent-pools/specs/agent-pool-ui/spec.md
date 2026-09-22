# Spec Delta

## MODIFIED Requirements

### Requirement: Modal de configuration et de lancement du pool
L'interface utilisateur SHALL inclure, dans le Kanban de chaque workspace, un bouton de lancement d'action qui ouvre une modale de configuration du Pool d'Agents pour ce workspace. Cette modale SHALL permettre à l'utilisateur d'ajuster le nombre de workers parallèles (de 1 à 5) et de sélectionner le Mode de Délégation (`full-autonomy` ou `hitl-review`). L'état affiché par ce bouton (actif ou inactif) SHALL toujours refléter le statut réel du pool de ce workspace tel que connu du backend, y compris juste après le chargement de la page ou un changement de workspace, plutôt qu'un état mémorisé localement dans l'interface.

#### Scenario: Ouverture et configuration de la modale
- **WHEN** l'utilisateur clique sur le bouton "Lancer le Pool" alors qu'aucun pool n'est actif sur ce workspace
- **THEN** une modale s'affiche avec un sélecteur numérique pour la taille du pool, un commutateur pour le mode de délégation, et un bouton de confirmation "Démarrer"

#### Scenario: Validation et lancement du pool
- **WHEN** l'utilisateur clique sur "Démarrer" dans la modale de configuration
- **THEN** l'application envoie les paramètres au backend via l'API pour ce workspace, ferme la modale, et affiche un indicateur visuel de pool actif dans l'en-tête du Kanban de ce workspace

#### Scenario: Statut du bouton synchronisé au chargement de la page
- **WHEN** l'utilisateur ouvre ou recharge le Kanban d'un workspace pour lequel un pool est déjà actif côté backend
- **THEN** le bouton affiche immédiatement l'état "pool actif" pour ce workspace, sans nécessiter d'action de l'utilisateur

#### Scenario: Statut du bouton correct après changement de workspace
- **WHEN** l'utilisateur navigue du Kanban du workspace `A` (pool actif) vers le Kanban du workspace `B` (aucun pool actif)
- **THEN** le bouton du workspace `B` affiche l'état "aucun pool actif", indépendamment de l'état affiché précédemment pour le workspace `A`
