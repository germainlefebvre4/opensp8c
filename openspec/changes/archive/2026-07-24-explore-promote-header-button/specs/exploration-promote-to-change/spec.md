## ADDED Requirements

### Requirement: Bouton de promotion dans le header d'exploration anonyme
Le frontend SHALL afficher un bouton d'action discret "Créer le change" dans le header du volet d'exploration anonyme (`ExploreAnonymousPanel`) dès que l'exploration a démarré (présence de `ghostId` et `ghostName`).

#### Scenario: Bouton affiché dès le démarrage de l'exploration anonyme
- **WHEN** le volet d'exploration anonyme est ouvert et dispose de `ghostId` et `ghostName`
- **THEN** un bouton d'action "Créer le change" s'affiche dans le header à côté des actions système (agrandir/fermer)

#### Scenario: Bouton absent si l'exploration n'est pas initialisée ou déjà associée à un change
- **WHEN** le volet d'exploration s'ouvre pour une exploration déjà associée à un change normal (non-ghost)
- **THEN** le bouton de promotion n'est pas affiché dans le header

### Requirement: Comportement responsive du bouton de promotion
Le bouton de promotion dans le header SHALL adapter sa présentation selon la largeur du volet d'exploration.

#### Scenario: Largeur suffisante — texte et icône
- **WHEN** le volet d'exploration a une largeur supérieure ou égale à 350px
- **THEN** le bouton de promotion affiche une icône d'action (`✨`) suivie du texte descriptif (ex: "Créer le change")

#### Scenario: Largeur étroite — icône seule avec tooltip
- **WHEN** le volet d'exploration a une largeur inférieure à 350px
- **THEN** le bouton de promotion n'affiche que l'icône `✨` et affiche un tooltip descriptif lors du survol (ex: "Créer le change à partir de cette exploration")

### Requirement: Dialogue de confirmation de promotion depuis le volet
Le clic sur le bouton de promotion du volet d'exploration SHALL ouvrir la même boîte de dialogue de confirmation que le drag-and-drop, permettant de modifier le nom et de confirmer ou annuler la promotion.

#### Scenario: Validation de la promotion depuis le dialogue
- **WHEN** l'utilisateur clique sur le bouton de promotion, confirme/modifie le nom du change dans le dialogue, et clique sur [Créer le change]
- **THEN** le dialogue et le volet d'exploration se ferment, la carte correspondante passe en état "FF running", et l'appel API `POST /api/workspaces/{id}/explorations/{ghostId}/promote` est envoyé
