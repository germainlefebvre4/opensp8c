## ADDED Requirements

### Requirement: Activer le mode édition d'une spec
L'utilisateur SHALL pouvoir basculer en mode édition sur une spec sélectionnée via un bouton "Éditer" visible dans la vue de contenu.

#### Scenario: Entrée en mode édition
- **WHEN** l'utilisateur clique sur le bouton "Éditer" d'une spec affichée
- **THEN** la vue passe en mode édition avec un split view [textarea | panneau diff]
- **AND** la textarea contient le contenu actuel de la spec
- **AND** le panneau diff est vide (aucune modification en cours)
- **AND** les boutons "Enregistrer" et "Annuler" sont visibles

#### Scenario: Le bouton Éditer n'est visible qu'avec une spec sélectionnée
- **WHEN** aucune spec n'est sélectionnée
- **THEN** aucun bouton "Éditer" n'est affiché

### Requirement: Édition du contenu Markdown dans la textarea
En mode édition, l'utilisateur SHALL pouvoir modifier librement le contenu Markdown de la spec dans une textarea.

#### Scenario: Modification du contenu
- **WHEN** l'utilisateur modifie le texte dans la textarea
- **THEN** les modifications sont reflétées immédiatement dans la textarea
- **AND** le panneau diff se met à jour pour montrer les différences avec le contenu sauvegardé

#### Scenario: Contenu identique à l'original
- **WHEN** l'utilisateur modifie puis revert manuellement ses changements
- **THEN** le panneau diff revient à l'état vide (aucune modification)

### Requirement: Affichage du diff live
En mode édition, le panneau droit SHALL afficher en temps réel les différences ligne par ligne entre le contenu actuel de la spec (source de vérité disque) et le contenu en cours de modification.

#### Scenario: Lignes ajoutées
- **WHEN** l'utilisateur ajoute des lignes dans la textarea
- **THEN** le panneau diff affiche ces lignes avec un fond vert et le préfixe `+`

#### Scenario: Lignes supprimées
- **WHEN** l'utilisateur supprime des lignes dans la textarea
- **THEN** le panneau diff affiche ces lignes avec un fond rouge et le préfixe `-`

#### Scenario: Lignes inchangées
- **WHEN** des lignes n'ont pas été modifiées
- **THEN** elles apparaissent dans le panneau diff avec un style neutre sans préfixe

#### Scenario: Aucune modification
- **WHEN** le contenu de la textarea est identique au contenu sauvegardé
- **THEN** le panneau diff affiche un état vide ou un message "Aucune modification"

### Requirement: Enregistrement explicite d'une spec
L'utilisateur SHALL pouvoir enregistrer les modifications via le bouton "Enregistrer" ou le raccourci Ctrl+S. La spec modifiée est écrite sur disque ; le fichier est la source de vérité.

#### Scenario: Save via bouton
- **WHEN** l'utilisateur clique sur "Enregistrer"
- **THEN** le contenu de la textarea est envoyé via PUT au backend
- **AND** le fichier `spec.md` est mis à jour sur disque
- **AND** le watcher détecte la modification et émet un événement SSE `spec_updated`
- **AND** l'UI invalide son cache et relit la spec depuis le disque
- **AND** le panneau diff revient à l'état vide

#### Scenario: Save via Ctrl+S
- **WHEN** l'utilisateur appuie sur Ctrl+S en mode édition
- **THEN** le comportement est identique au save via bouton

#### Scenario: Echec de l'enregistrement
- **WHEN** le backend retourne une erreur lors du PUT
- **THEN** un message d'erreur est affiché
- **AND** la textarea conserve son contenu (aucune perte de données)
- **AND** le mode édition reste actif

### Requirement: Annulation de l'édition
L'utilisateur SHALL pouvoir annuler l'édition via le bouton "Annuler", ce qui retourne à la vue lecture sans sauvegarder.

#### Scenario: Annulation sans modifications
- **WHEN** l'utilisateur clique sur "Annuler" sans avoir modifié le contenu
- **THEN** la vue retourne en mode lecture avec le contenu original

#### Scenario: Annulation avec modifications non sauvegardées
- **WHEN** l'utilisateur clique sur "Annuler" après avoir modifié le contenu
- **THEN** la vue retourne en mode lecture avec le contenu original (modifications perdues)

### Requirement: Détection de modification externe en cours d'édition
Si un fichier `spec.md` est modifié en dehors de l'UI (CLI, éditeur externe) pendant qu'il est en cours d'édition, l'UI SHALL en informer l'utilisateur sans écraser automatiquement ses modifications locales.

#### Scenario: Modification externe détectée
- **WHEN** l'UI est en mode édition sur une spec
- **AND** un événement SSE `spec_updated` arrive pour cette spec
- **AND** le contenu local diffère du contenu récupéré
- **THEN** un banner d'avertissement est affiché : "Ce fichier a été modifié en dehors de l'éditeur"
- **AND** deux actions sont proposées : "Ignorer" (conserver les modifications locales) et "Recharger" (écraser avec la version disque)

#### Scenario: Modification externe mais contenu identique
- **WHEN** un événement SSE `spec_updated` arrive
- **AND** le contenu distant est identique au contenu local
- **THEN** aucun avertissement n'est affiché

### Requirement: Synchronisation en temps réel des specs via SSE
L'application SHALL écouter les événements SSE `spec_updated` pour invalider automatiquement le cache d'une spec et recharger son contenu depuis le disque.

#### Scenario: Réception d'un événement spec_updated en mode lecture
- **WHEN** un événement SSE `spec_updated` est reçu pour une spec
- **AND** l'UI est en mode lecture sur cette spec
- **THEN** le contenu de la spec est automatiquement rechargé depuis le serveur

#### Scenario: Réception d'un événement spec_updated pour une autre spec
- **WHEN** un événement SSE `spec_updated` est reçu pour une spec non sélectionnée
- **THEN** le cache de cette spec est invalidé mais aucun re-fetch immédiat n'est déclenché
