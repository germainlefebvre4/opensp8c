# Spec Delta

## ADDED Requirements

### Requirement: Action « Reprendre » sur les workers en pause
Le panneau d'état du pool actif SHALL afficher, pour chaque worker au statut `paused`, sa raison de blocage lisible et un bouton « Reprendre » (« Resume » en anglais). Un clic sur ce bouton SHALL demander au backend la reprise de ce seul worker, sans arrêter le pool ni interrompre les autres workers. Pendant la requête, le bouton SHALL être désactivé pour éviter une double demande. Si la reprise échoue, le panneau SHALL afficher l'erreur retournée par le backend et conserver le worker en pause. Le bouton SHALL NOT apparaître pour les workers dans un autre statut.

#### Scenario: Bouton visible sur un worker en pause
- **WHEN** l'utilisateur ouvre le panneau d'état alors que le worker 1, sur le changement `add-user-auth`, est `paused`
- **THEN** la ligne de ce worker affiche sa raison de blocage et un bouton « Reprendre »

#### Scenario: Reprise d'un worker
- **WHEN** l'utilisateur clique sur « Reprendre » pour le worker en pause du changement `add-user-auth`
- **THEN** l'application envoie la demande de reprise pour ce worker, le panneau reste ouvert, et la ligne reflète le nouveau statut du worker dès sa reprise par le pool, sans rechargement de la page

#### Scenario: Aucun bouton hors pause
- **WHEN** un worker est au statut `working`, `testing` ou `healing`
- **THEN** sa ligne n'affiche pas de bouton « Reprendre »

#### Scenario: Échec de la reprise
- **WHEN** le backend refuse la reprise, par exemple parce que le worker n'est plus en pause
- **THEN** le panneau affiche le message d'erreur et le bouton redevient actif
