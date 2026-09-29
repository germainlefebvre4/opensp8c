# Explore Message Layout Specification

## Purpose

Remplace le rendu en bulles de conversation par un flux de lecture plein-largeur pour les tours utilisateur et assistant, et rend visibles les appels d'outils de l'agent sous forme de lignes repliables plutôt que de les masquer.

## Requirements

### Requirement: Tours de conversation en pleine largeur

Les tours utilisateur et assistant SHALL occuper toute la largeur disponible du fil de conversation, sans alignement gauche/droite ni fond de bulle coloré. Chaque tour SHALL porter un léger indicateur de rôle (libellé et/ou icône) plutôt qu'un positionnement gauche/droite pour distinguer utilisateur et assistant.

#### Scenario: Rendu d'un tour utilisateur
- **WHEN** un message dont le rôle est `user` est affiché dans le fil
- **THEN** son texte occupe toute la largeur disponible du fil, précédé d'un indicateur de rôle, sans fond de bulle coloré ni alignement à droite

#### Scenario: Rendu d'un tour assistant
- **WHEN** un message dont le rôle est `assistant` est affiché dans le fil
- **THEN** son texte occupe toute la largeur disponible du fil, précédé d'un indicateur de rôle, sans fond de bulle coloré ni alignement à gauche

### Requirement: Capture des appels d'outils dans le flux de messages

Le frontend SHALL capturer les blocs `tool_use` (et le `tool_result` associé) produits par le subprocess agent, au lieu de les filtrer comme aujourd'hui, et SHALL les rattacher au message assistant dont le texte les suit dans le flux.

#### Scenario: Capture d'un bloc tool_use
- **WHEN** le subprocess agent produit un bloc de contenu de type `tool_use` suivi de texte assistant
- **THEN** l'appel d'outil est conservé (nom de l'outil et cible/entrée) et associé au message assistant contenant ce texte, au lieu d'être filtré

#### Scenario: Résultat d'outil disponible pour dépliage
- **WHEN** un `tool_result` correspondant à un `tool_use` capturé est reçu
- **THEN** un aperçu du résultat est conservé avec l'appel d'outil, disponible pour affichage au dépliage de la ligne

#### Scenario: Message assistant sans appel d'outil
- **WHEN** un message assistant ne contient aucun bloc `tool_use` en amont de son texte
- **THEN** aucune ligne d'appel d'outil n'est affichée au-dessus de ce message

### Requirement: Affichage replié d'un appel d'outil

Chaque appel d'outil capturé SHALL être affiché comme une ligne compacte unique (icône, nom de l'outil, cible) au-dessus du texte du message assistant qu'il précède, repliée par défaut. Le JSON brut de l'appel ne SHALL JAMAIS être affiché.

#### Scenario: Ligne repliée par défaut
- **WHEN** un message assistant précédé d'un ou plusieurs appels d'outils est affiché pour la première fois
- **THEN** chaque appel apparaît comme une ligne compacte repliée (icône + nom de l'outil + cible), sans détail du résultat visible

#### Scenario: Dépliage d'une ligne d'appel d'outil
- **WHEN** l'utilisateur clique sur une ligne d'appel d'outil repliée
- **THEN** un aperçu du résultat associé s'affiche sous la ligne, sans jamais afficher le JSON brut de l'appel ou du résultat

#### Scenario: Repliage d'une ligne dépliée
- **WHEN** l'utilisateur clique sur une ligne d'appel d'outil actuellement dépliée
- **THEN** l'aperçu du résultat se referme et la ligne reprend son état compact

### Requirement: Cohérence du flux plein-largeur entre panneaux et relecture

Le flux plein-largeur et l'affichage des appels d'outils SHALL être rendus de façon identique dans `ExploreAnonymousPanel`, `ExplorePanel`, et l'onglet de relecture en lecture seule de `DetailPanel`.

#### Scenario: Cohérence entre panel anonyme et panel nommé
- **WHEN** une conversation est affichée successivement dans un panel d'exploration anonyme puis dans un panel nommé pour le même historique
- **THEN** les tours et les appels d'outils sont rendus avec la même mise en page plein-largeur dans les deux panels

#### Scenario: Relecture en lecture seule
- **WHEN** l'onglet de relecture (log) de `DetailPanel` affiche l'historique d'une conversation passée
- **THEN** les tours et les appels d'outils capturés sont rendus avec le même flux plein-largeur que les panels de chat actifs, sans champ de saisie ni possibilité d'envoyer un message

### Requirement: Largeur de la carte de question

`QuestionCard` SHALL s'afficher sur toute la largeur disponible du flux plein-largeur, tout en conservant son style actuel (fond blanc, bordure et ombre légère) qui la distingue des tours de texte ordinaires.

#### Scenario: Affichage d'une question dans le flux plein-largeur
- **WHEN** une question de clarification est affichée dans un fil utilisant le flux plein-largeur
- **THEN** la carte de question occupe toute la largeur disponible du fil, avec son fond blanc, sa bordure et son ombre légère inchangés

### Requirement: Notice système dans le flux plein-largeur
Le fil SHALL pouvoir afficher des notices système : lignes compactes, pleine largeur, en texte petit et discret avec une icône, sans indicateur de rôle utilisateur/assistant, sans fond de bulle et sans JSON brut. Une notice est conservée avec l'historique de la conversation.

#### Scenario: Rendu d'une notice
- **WHEN** une notice système est présente dans le fil
- **THEN** elle est rendue sur une ligne compacte (icône + texte) visuellement distincte d'un tour utilisateur ou assistant

#### Scenario: Cohérence entre panels et relecture
- **WHEN** une conversation contenant une notice est affichée dans `ExploreAnonymousPanel` ou `ExplorePanel`
- **THEN** la notice est rendue de la même façon dans les deux

#### Scenario: Onglet conversation de DetailPanel
- **WHEN** l'onglet conversation de `DetailPanel` est affiché
- **THEN** il reste une timeline d'activités issue des logs et ne rend pas de notice : elle ne consomme pas de messages de chat

#### Scenario: Mode de rendu brut/markdown
- **WHEN** l'utilisateur bascule entre le mode brut et le mode rendu
- **THEN** la notice conserve le même aspect dans les deux modes
