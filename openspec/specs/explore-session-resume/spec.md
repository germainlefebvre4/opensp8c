# Spec: Explore Session Resume

## Purpose

TBD — Capabilities for persisting and resuming Claude session identifiers across explore session lifecycle events (subprocess restarts, inactivity timeouts).

## Requirements

### Requirement: Persistance de l'identifiant de session Claude pour les named sessions
Le backend SHALL générer un UUID à la première ouverture d'une named session et le persister dans `preferences.json` sous la clé `workspaceID/changeName`, au sein du champ `claudeSessionId`.

#### Scenario: Première ouverture d'une named session
- **WHEN** `Manager.Start` est appelé pour un changeName sans `claudeSessionId` existant dans preferences
- **THEN** un UUID est généré, le subprocess est lancé avec `--session-id <uuid>`, et l'UUID est immédiatement stocké dans preferences.json avant que le subprocess ne réponde

#### Scenario: Réouverture d'une named session existante
- **WHEN** `Manager.Start` est appelé pour un changeName ayant déjà un `claudeSessionId` dans preferences
- **THEN** le subprocess est lancé avec `--resume <claudeSessionId>` (pas de nouvel UUID généré)

#### Scenario: Session anonyme non persistée
- **WHEN** `Manager.StartAnonymous` est appelé
- **THEN** aucun UUID n'est généré ni stocké dans preferences.json — le comportement est inchangé

---

### Requirement: Reprise du contexte Claude après expiration du subprocess
Quand un subprocess est relancé pour une named session dont le `claudeSessionId` est connu, Claude SHALL reprendre le contexte conversationnel complet de la session précédente.

#### Scenario: Subprocess relancé après timeout d'inactivité
- **WHEN** le subprocess d'une named session a été tué par inactivité ET l'utilisateur rouvre le panneau
- **THEN** le nouveau subprocess est lancé avec `--resume <claudeSessionId>`, Claude reprend la conversation là où elle s'était arrêtée

#### Scenario: Fallback si --resume échoue
- **WHEN** le subprocess lancé avec `--resume <claudeSessionId>` produit une erreur au démarrage
- **THEN** le backend log un warning et relance un nouveau subprocess sans `--resume` ; le `claudeSessionId` dans preferences n'est pas supprimé

---

### Requirement: Non-injection du message initial sur session reprise
Le backend SHALL NOT envoyer le message d'initialisation `/opsx:explore <changeName>` lorsqu'une session est reprise via `--resume`.

#### Scenario: Reprise de session — pas d'injection
- **WHEN** `Manager.Start` est appelé avec un `claudeSessionId` existant dans preferences (session reprise)
- **THEN** le message `/opsx:explore <changeName>` n'est pas envoyé sur stdin du subprocess

#### Scenario: Première ouverture — injection normale
- **WHEN** `Manager.Start` est appelé sans `claudeSessionId` dans preferences (première ouverture)
- **THEN** le message `/opsx:explore <changeName>` est envoyé sur stdin du subprocess, comme actuellement
