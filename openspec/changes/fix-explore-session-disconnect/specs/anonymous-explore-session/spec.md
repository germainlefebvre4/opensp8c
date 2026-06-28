## MODIFIED Requirements

### Requirement: System prompt différencié pour session anonyme
Une session anonyme SHALL recevoir un system prompt additionnel qui instruit le LLM d'émettre le marqueur `change_created` après avoir créé un change, et qui désactive l'auto-injection de `/opsx:explore`. Au démarrage, le backend SHALL injecter un message d'amorce sur stdin du subprocess pour le maintenir actif et déclencher un message de bienvenue court vers l'utilisateur.

#### Scenario: System prompt anonyme injecté
- **WHEN** une session anonyme démarre
- **THEN** le subprocess reçoit `--append-system-prompt` avec la consigne d'émettre `{"event":"change_created","name":"..."}` après `/opsx:ff` ou `/opsx:new`

#### Scenario: Message d'amorce injecté au démarrage
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess
- **THEN** le backend écrit immédiatement un message user sur stdin du subprocess lui demandant de se présenter brièvement en une phrase pour inviter l'utilisateur à décrire son projet

#### Scenario: Pas d'auto-injection opsx:explore
- **WHEN** une session anonyme démarre
- **THEN** le backend n'envoie PAS `/opsx:explore <changeName>` sur stdin (contrairement aux sessions nommées)

#### Scenario: Subprocess reste actif en attendant la saisie utilisateur
- **WHEN** la session anonyme vient d'être créée et aucun message utilisateur n'a encore été saisi
- **THEN** le subprocess est toujours en cours d'exécution et le WebSocket reste connecté
