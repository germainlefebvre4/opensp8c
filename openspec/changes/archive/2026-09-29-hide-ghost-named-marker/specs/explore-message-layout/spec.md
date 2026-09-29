# Spec Delta

## ADDED Requirements

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
