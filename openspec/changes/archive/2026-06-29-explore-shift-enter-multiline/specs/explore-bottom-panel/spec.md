## ADDED Requirements

### Requirement: Saisie multiligne dans la conversation d'exploration
Le champ de saisie des panels d'exploration SHALL être un textarea auto-redimensionnable. L'utilisateur SHALL pouvoir insérer une nouvelle ligne avec `Shift+Enter`. La touche `Enter` seule SHALL envoyer le message sans insérer de nouvelle ligne.

#### Scenario: Shift+Enter insère une nouvelle ligne
- **WHEN** l'utilisateur appuie sur `Shift+Enter` dans le champ de saisie
- **THEN** une nouvelle ligne est insérée dans le message et le message n'est pas envoyé

#### Scenario: Enter seul envoie le message
- **WHEN** l'utilisateur appuie sur `Enter` sans maintenir `Shift`
- **THEN** le message est envoyé et le champ de saisie est vidé

#### Scenario: Auto-resize du textarea
- **WHEN** l'utilisateur saisit du texte sur plusieurs lignes
- **THEN** le textarea augmente en hauteur pour afficher tout le contenu sans scrollbar interne, dans la limite d'une hauteur maximale

#### Scenario: Reset de la hauteur après envoi
- **WHEN** le message est envoyé et le champ de saisie est vidé
- **THEN** le textarea reprend sa hauteur initiale (une ligne)
