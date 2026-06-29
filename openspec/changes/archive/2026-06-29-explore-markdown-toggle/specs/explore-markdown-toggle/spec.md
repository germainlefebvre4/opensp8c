## ADDED Requirements

### Requirement: Toggle raw/rendered dans le header du panel d'exploration
Le panel d'exploration SHALL afficher un toggle permettant de basculer entre le mode `raw` (texte brut) et le mode `rendered` (markdown interprété) pour les messages de l'assistant.

#### Scenario: Toggle visible dans le header
- **WHEN** un panel d'exploration est ouvert (anonyme ou nommé)
- **THEN** le header affiche deux boutons icône (Code pour raw, Eye pour rendered) dans un container groupé

#### Scenario: Mode raw actif par défaut
- **WHEN** aucune préférence n'est sauvegardée en localStorage
- **THEN** le mode `raw` est actif et le bouton Code est mis en évidence

#### Scenario: Basculer en mode rendered
- **WHEN** l'utilisateur clique sur le bouton Eye
- **THEN** les messages assistant existants et futurs sont affichés avec le rendu markdown interprété

#### Scenario: Basculer en mode raw
- **WHEN** l'utilisateur clique sur le bouton Code alors que le mode rendered est actif
- **THEN** les messages assistant sont affichés en texte brut avec whitespace-pre-wrap

### Requirement: Persistance de la préférence en localStorage
La préférence de mode raw/rendered SHALL être persistée dans le localStorage sous la clé `explore-view-mode`.

#### Scenario: Sauvegarde automatique lors du changement
- **WHEN** l'utilisateur bascule le toggle
- **THEN** la nouvelle valeur (`raw` ou `rendered`) est écrite dans localStorage sous `explore-view-mode`

#### Scenario: Restauration au rechargement
- **WHEN** l'utilisateur recharge la page et ouvre un panel d'exploration
- **THEN** le mode précédemment choisi est restauré depuis localStorage

#### Scenario: Fallback si localStorage indisponible
- **WHEN** le localStorage est inaccessible (mode privé, storage bloqué)
- **THEN** le panel s'initialise en mode `raw` sans erreur

### Requirement: Rendu conditionnel des messages selon le mode
Les messages de l'assistant SHALL être rendus différemment selon le mode actif ; les messages utilisateur SHALL toujours être rendus en texte brut.

#### Scenario: Messages assistant en mode rendered
- **WHEN** le mode `rendered` est actif
- **THEN** le contenu des messages assistant est passé à ReactMarkdown avec les classes `prose prose-slate prose-sm max-w-none`

#### Scenario: Messages assistant en mode raw
- **WHEN** le mode `raw` est actif
- **THEN** le contenu des messages assistant est affiché avec `whitespace-pre-wrap` (comportement actuel)

#### Scenario: Messages utilisateur toujours en raw
- **WHEN** le mode `rendered` est actif
- **THEN** les messages dont le rôle est `user` continuent d'être rendus en texte brut (bg-blue-600)

#### Scenario: Messages partiels rendus en mode rendered
- **WHEN** le mode `rendered` est actif et qu'un message assistant est en cours de streaming (`partial: true`)
- **THEN** le contenu partiel est passé à ReactMarkdown (rendu potentiellement instable accepté)

### Requirement: Cohérence du toggle entre panels anonyme et nommé
Le toggle SHALL fonctionner de manière identique dans `ExploreAnonymousPanel` et `ExplorePanel`, partageant la même clé localStorage.

#### Scenario: Préférence partagée entre panels
- **WHEN** l'utilisateur change le mode dans un panel anonyme puis ouvre un panel nommé
- **THEN** le panel nommé s'initialise avec la même préférence
