# Spec Delta

## MODIFIED Requirements

### Requirement: Ouvrir une session d'exploration
L'utilisateur SHALL pouvoir ouvrir une session de chat avec Claude Code depuis une carte en colonne **To Explore**. Le backend SHALL lancer un subprocess `claude` long-lived avec les flags appropriés pour un chat non-interactif, dans le répertoire racine du workspace actif, avec un prompt système d'exploration incluant le cadrage progressif et le marqueur de question (voir requirement "Prompt système : cadrage progressif et marqueur de question").

#### Scenario: Ouverture de la session
- **WHEN** l'utilisateur clique sur le bouton "Explorer" d'une carte en colonne To Explore
- **THEN** un panneau de chat s'ouvre et le backend spawn un subprocess `claude` avec `--print --input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt "<prompt système d'exploration>" --cwd <workspace-root>`, sans qu'aucun message d'accueil statique ne soit affiché en amont côté frontend

#### Scenario: Session déjà ouverte pour ce changement
- **WHEN** l'utilisateur clique sur "Explorer" pour un changement dont une session est déjà active
- **THEN** le panneau de chat existant est affiché (pas de nouveau subprocess spawné)

### Requirement: Interdire les interactions à choix multiples
Le subprocess agent SHALL être configuré par défaut pour ne jamais produire de questions à choix multiples ou utiliser l'outil `AskUserQuestion`. Seuls le chat textuel et le marqueur de question conventionnel (`ghost_question`, voir `explore-structured-questions`) sont autorisés par défaut. Une exception SHALL exister uniquement lorsque l'agent actif est Claude ET que le réglage global du mode question native (`explore-native-question-mode`) est activé : dans ce cas, l'outil `AskUserQuestion` SHALL être autorisé à la place de l'interdiction.

#### Scenario: Bloc AskUserQuestion ignoré par défaut
- **WHEN** le subprocess tente de produire une interaction `AskUserQuestion` et que le mode question native n'est pas effectif pour cette session
- **THEN** le système prompt injecté (`--append-system-prompt`) prévient ce comportement et l'agent répond par du texte libre ou un marqueur `ghost_question` à la place

#### Scenario: Interdiction levée en mode question native
- **WHEN** l'agent actif est Claude et le réglage global du mode question native est activé pour cette session
- **THEN** le système prompt injecté n'interdit pas l'outil `AskUserQuestion`

## ADDED Requirements

### Requirement: Prompt système : cadrage progressif et marqueur de question

Le prompt système d'exploration (base pour les sessions nommées, `anonSystemPrompt` pour les sessions anonymes) SHALL instruire l'agent à consacrer sa toute première réponse au cadrage de l'exploration (compréhension de l'intention, questions ouvertes nécessaires) plutôt qu'à une réponse générique, et à signaler toute question de clarification, y compris celles de ce premier tour, via le marqueur `ghost_question` défini par `explore-structured-questions`.

#### Scenario: Premier tour d'une session anonyme
- **WHEN** l'utilisateur envoie son premier message dans une exploration anonyme
- **THEN** la réponse de l'agent vise le cadrage de l'exploration et utilise le marqueur `ghost_question` pour toute question de clarification posée

#### Scenario: Premier tour d'une session nommée
- **WHEN** une named session démarre pour la première fois (message `/opsx:explore <changeName>` auto-injecté)
- **THEN** le prompt système appliqué à cette session porte les mêmes instructions de cadrage progressif et de marqueur de question que pour une session anonyme
