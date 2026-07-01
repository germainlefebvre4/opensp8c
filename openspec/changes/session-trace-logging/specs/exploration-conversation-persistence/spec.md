## ADDED Requirements

### Requirement: Réutilisation de l'identité du ghost à la reprise backend

Quand le frontend reprend une exploration dont la session backend a expiré, il SHALL transmettre l'id du ghost au backend via `POST /explore/sessions`. Le backend SHALL, si cet id correspond à une exploration existante dans le workspace, réutiliser cet id comme identifiant de la nouvelle session anonyme au lieu d'en générer un nouveau.

#### Scenario: Reprise transmet l'id du ghost
- **WHEN** le frontend ouvre une session d'exploration avec un `resumeGhostId` connu
- **THEN** `POST /api/workspaces/{id}/explore/sessions` est appelé avec `{"resumeGhostId": "<ghostId>"}` dans le corps de la requête

#### Scenario: Id de reprise valide — réutilisation
- **WHEN** le backend reçoit un `resumeGhostId` correspondant à une exploration existante dans `preferences.json` pour ce workspace
- **THEN** la nouvelle session anonyme est créée (ou rattachée si encore vivante) sous ce même id, et non un id généré aléatoirement

#### Scenario: Id de reprise invalide ou inconnu — fallback silencieux
- **WHEN** le backend reçoit un `resumeGhostId` qui ne correspond à aucune exploration du workspace (supprimée entre-temps, id erroné)
- **THEN** le backend ignore silencieusement le `resumeGhostId` et démarre une session anonyme avec un id généré normalement, sans erreur retournée au frontend

#### Scenario: Session encore vivante — pas de nouveau subprocess
- **WHEN** un `resumeGhostId` valide correspond à une session encore active dans `session.Manager` (non expirée)
- **THEN** le backend rattache la connexion à cette session existante plutôt que de démarrer un nouveau subprocess

#### Scenario: Session expirée — nouveau subprocess sous le même id
- **WHEN** un `resumeGhostId` valide ne correspond à aucune session active dans `session.Manager` (expirée)
- **THEN** le backend démarre un nouveau subprocess dont l'id de session est le `resumeGhostId`, faisant atterrir son log de conversation dans le même dossier que la session d'origine
