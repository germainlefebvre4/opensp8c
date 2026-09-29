# Spec Delta

## ADDED Requirements

### Requirement: Le marker ghost_named n'est jamais affiché dans le fil
Le marker `{"event":"ghost_named","name":"<name>"}` SHALL être retiré du texte assistant affiché, quelle que soit la façon dont il arrive : sur sa propre ligne, mélangé à du texte, dans un message consolidé, ou fragmenté sur plusieurs deltas de streaming. Le texte environnant SHALL être conservé. Aucun fragment de JSON du marker (par exemple `{"`, `event"`) SHALL apparaître, même transitoirement, dans un tour assistant.

#### Scenario: Marker fragmenté sur plusieurs deltas
- **WHEN** l'agent streame le marker en deltas successifs (`{"`, `event"`, `":"ghost_named"`, …, `}\n\n`) suivis de texte ordinaire
- **THEN** aucun de ces fragments n'est affiché et seul le texte ordinaire apparaît dans le tour assistant

#### Scenario: Marker dans un message consolidé
- **WHEN** le message de fin de tour contient le marker suivi de texte
- **THEN** le texte est affiché sans le marker et sans ligne vide superflue en tête

#### Scenario: Accolade légitime en début de réponse
- **WHEN** la réponse commence par un `{` qui ne mène pas à un marker connu (ex. un exemple de code JSON)
- **THEN** le texte retenu en attente est restitué intégralement et dans l'ordre dès que l'invalidation est certaine

### Requirement: Notice de nommage dans le fil
Lorsque le nom du ghost card est attribué, le fil de l'exploration SHALL afficher une notice système unique « Exploration nommée : `<nom final>` », dont le nom est celui effectivement retenu (suffixe de collision inclus).

#### Scenario: Nommage de l'exploration
- **WHEN** l'événement `ghost_named` est reçu avec le nom `color-palette-primary-success-distinction`
- **THEN** une notice « Exploration nommée : `color-palette-primary-success-distinction` » est insérée dans le fil, et le header et la carte kanban affichent ce nom

#### Scenario: Nom modifié par collision
- **WHEN** le nom proposé est déjà pris et le backend retient `<nom>-2`
- **THEN** la notice affiche `<nom>-2`

#### Scenario: Reprise de session
- **WHEN** une exploration déjà nommée est rouverte ou l'historique est rechargé
- **THEN** la notice de nommage apparaît une seule fois, sans doublon
