## MODIFIED Requirements

### Requirement: Auto-injection du message initial d'exploration
Au démarrage d'un nouveau subprocess pour une session d'exploration **nommée**, le backend SHALL envoyer automatiquement `/opsx:explore <changeName>` comme premier message stdin avant tout message utilisateur. Les sessions anonymes SHALL NOT recevoir ce message d'initialisation.

#### Scenario: Injection au démarrage du subprocess (session nommée)
- **WHEN** `Manager.Start` démarre un nouveau subprocess pour un changement dont le nom est connu
- **THEN** le backend écrit immédiatement `{"type":"user","message":{"role":"user","content":"/opsx:explore <changeName>"}}\n` sur le stdin du subprocess

#### Scenario: Pas de double injection sur session existante
- **WHEN** `Manager.Start` est appelé pour un changement dont une session est déjà active (subprocess en cours)
- **THEN** aucun message d'injection n'est envoyé et la session existante est retournée telle quelle

#### Scenario: Pas d'injection pour session anonyme
- **WHEN** `Manager.StartAnonymous` démarre un nouveau subprocess sans changeName connu
- **THEN** aucun message `/opsx:explore` n'est envoyé sur stdin du subprocess
