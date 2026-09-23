# Spec Delta

## MODIFIED Requirements

### Requirement: Rendu conditionnel des messages selon le mode
Les messages de l'assistant SHALL être rendus différemment selon le mode actif ; les messages utilisateur SHALL toujours être rendus en texte brut. Ce rendu s'applique au contenu du tour plein-largeur (`explore-message-layout`), sans lien avec un fond de bulle.

#### Scenario: Messages assistant en mode rendered
- **WHEN** le mode `rendered` est actif
- **THEN** le contenu des messages assistant est passé à ReactMarkdown avec les classes `prose prose-slate prose-sm max-w-none`

#### Scenario: Messages assistant en mode raw
- **WHEN** le mode `raw` est actif
- **THEN** le contenu des messages assistant est affiché avec `whitespace-pre-wrap` (comportement actuel)

#### Scenario: Messages utilisateur toujours en raw
- **WHEN** le mode `rendered` est actif
- **THEN** les messages dont le rôle est `user` continuent d'être rendus en texte brut, dans le tour plein-largeur qui leur est propre (sans fond de bulle)

#### Scenario: Messages partiels rendus en mode rendered
- **WHEN** le mode `rendered` est actif et qu'un message assistant est en cours de streaming (`partial: true`)
- **THEN** le contenu partiel est passé à ReactMarkdown (rendu potentiellement instable accepté)
