## MODIFIED Requirements

### Requirement: System prompt différencié pour session anonyme
Une session anonyme SHALL recevoir un system prompt additionnel qui instruit le LLM d'émettre le marqueur `change_created` après avoir créé un change, et qui désactive l'auto-injection de `/opsx:explore`. Au démarrage, le backend SHALL ne plus injecter de message d'amorce sur stdin du subprocess ; l'UI affiche un message statique à la place.

#### Scenario: System prompt anonyme injecté
- **WHEN** une session anonyme démarre
- **THEN** le subprocess reçoit `--append-system-prompt` avec la consigne d'émettre `{"event":"change_created","name":"..."}` après `/opsx:ff` ou `/opsx:new`

#### Scenario: Pas de message d'amorce injecté au démarrage
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess
- **THEN** le backend n'écrit aucun message sur stdin du subprocess au démarrage

#### Scenario: Pas d'auto-injection opsx:explore
- **WHEN** une session anonyme démarre
- **THEN** le backend n'envoie PAS `/opsx:explore <changeName>` sur stdin (contrairement aux sessions nommées)

#### Scenario: Subprocess reste actif en attendant la saisie utilisateur
- **WHEN** la session anonyme vient d'être créée et aucun message utilisateur n'a encore été saisi
- **THEN** le subprocess est toujours en cours d'exécution et le WebSocket reste connecté
