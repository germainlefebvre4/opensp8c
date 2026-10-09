## MODIFIED Requirements

### Requirement: Onglet Agent Pool dans Configuration
Le système SHALL exposer, dans Configuration, un onglet nommé "Agent Pool" à la place de l'ancien onglet "Agents" (dont le contenu de registre est déplacé dans l'onglet "CLI"). Cet onglet SHALL contenir uniquement les réglages par défaut du pool d'agents ; la vue de visibilité globale des pools actifs SHALL NE PAS y figurer : elle est exposée par l'onglet « Agents » de la barre principale, décrit par la capacité `agent-pool-visibility`.

#### Scenario: Onglet renommé
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'onglet précédemment nommé "Agents" est désormais nommé "Agent Pool"
- **THEN** le registre des CLI installés ne s'affiche plus sous cet onglet

#### Scenario: Contenu de l'onglet Agent Pool
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool
- **THEN** seuls les réglages par défaut du pool (taille, mode de délégation, tentatives, commande de validation) s'affichent
- **THEN** aucune liste de pools ou de workers actifs n'est affichée

## REMOVED Requirements

### Requirement: Défauts de l'Agent Pool dans Configuration
**Reason**: Remplacée par « Réglages par défaut de l'Agent Pool dans Configuration » : la vue de visibilité des pools actifs n'est plus affichée sous les réglages.

**Migration**: Voir l'exigence « Réglages par défaut de l'Agent Pool dans Configuration ».

## ADDED Requirements

### Requirement: Réglages par défaut de l'Agent Pool dans Configuration
Le système SHALL exposer, dans Configuration > Agent Pool, des réglages par défaut du pool d'agents : taille (de 1 à 5), mode de délégation (`full-autonomy` ou `hitl-review`) et nombre maximal de tentatives d'auto-correction. Ces valeurs SHALL être persistées dans `preferences.json`, retournées par `GET /api/preferences` et servir de base à tous les workspaces, qui peuvent les surcharger dans Settings > Agent Pool. À défaut de réglage enregistré, la taille vaut 3, le mode `hitl-review` et le nombre de tentatives 3.

#### Scenario: Valeurs par défaut initiales
- **WHEN** l'utilisateur ouvre Configuration > Agent Pool sans réglage de pool enregistré
- **THEN** les défauts affichés sont une taille de 3, le mode `hitl-review` et 3 tentatives

#### Scenario: Enregistrement des défauts
- **WHEN** l'utilisateur fixe la taille à 2 et enregistre
- **THEN** `preferences.json` reflète ce défaut et `GET /api/preferences` le retourne
- **THEN** le pool démarré pour un workspace sans surcharge a une taille de 2

#### Scenario: Valeur invalide
- **WHEN** l'utilisateur tente d'enregistrer une taille de 0
- **THEN** l'enregistrement est refusé avec une erreur de validation et les défauts enregistrés ne sont pas modifiés
