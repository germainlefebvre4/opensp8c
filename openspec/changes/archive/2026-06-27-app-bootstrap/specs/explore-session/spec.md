## ADDED Requirements

### Requirement: Ouvrir une session d'exploration
L'utilisateur SHALL pouvoir ouvrir une session de chat avec Claude Code depuis une carte en colonne **To Explore**. Le backend SHALL lancer un subprocess `claude` long-lived avec les flags appropriés pour un chat non-interactif, dans le répertoire racine du workspace actif.

#### Scenario: Ouverture de la session
- **WHEN** l'utilisateur clique sur le bouton "Explorer" d'une carte en colonne To Explore
- **THEN** un panneau de chat s'ouvre et le backend spawn un subprocess `claude` avec `--print --input-format stream-json --output-format stream-json --include-partial-messages --append-system-prompt "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text." --cwd <workspace-root>`

#### Scenario: Session déjà ouverte pour ce changement
- **WHEN** l'utilisateur clique sur "Explorer" pour un changement dont une session est déjà active
- **THEN** le panneau de chat existant est affiché (pas de nouveau subprocess spawné)

### Requirement: Envoyer et recevoir des messages
L'utilisateur SHALL pouvoir envoyer des messages texte dans le chat. Les messages du subprocess SHALL être streamés en temps réel vers l'interface via WebSocket.

#### Scenario: Envoi d'un message utilisateur
- **WHEN** l'utilisateur soumet un message dans le champ de saisie
- **THEN** le message est transmis sur le stdin du subprocess via le backend, et affiché dans le fil de chat

#### Scenario: Réception d'une réponse streamée
- **WHEN** le subprocess produit des chunks de réponse sur stdout
- **THEN** chaque chunk est transmis au frontend via WebSocket et affiché en temps réel dans le fil de chat

### Requirement: Interdire les interactions à choix multiples
Le subprocess Claude SHALL être configuré pour ne jamais produire de questions à choix multiples ou utiliser l'outil `AskUserQuestion`. Seul le chat textuel est autorisé.

#### Scenario: Bloc AskUserQuestion ignoré
- **WHEN** le subprocess tente de produire une interaction `AskUserQuestion`
- **THEN** le système prompt injecté (`--append-system-prompt`) prévient ce comportement et Claude répond par du texte libre à la place

### Requirement: Fermer une session d'exploration
L'utilisateur SHALL pouvoir fermer le panneau de chat. La fermeture SHALL terminer proprement le subprocess.

#### Scenario: Fermeture du panneau
- **WHEN** l'utilisateur ferme le panneau de chat
- **THEN** le backend ferme le stdin du subprocess, attend sa terminaison propre, et libère les ressources associées

### Requirement: Timeout de session inactive
Une session d'exploration SHALL être automatiquement terminée après 30 minutes d'inactivité (aucun message envoyé ni reçu).

#### Scenario: Session expirée par inactivité
- **WHEN** aucun message n'a été échangé pendant 30 minutes
- **THEN** le subprocess est terminé, le panneau affiche un message "Session expirée — cliquez pour relancer" et propose de rouvrir une nouvelle session
