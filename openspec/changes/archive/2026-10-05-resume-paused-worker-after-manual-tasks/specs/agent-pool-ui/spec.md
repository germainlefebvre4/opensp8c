# Spec Delta

## MODIFIED Requirements

### Requirement: Action « Reprendre » sur les workers en pause
Le panneau d'état du pool actif SHALL afficher, pour chaque worker au statut `paused`, sa raison de blocage lisible et deux boutons : « Reprendre » (« Resume » en anglais) et « Reprendre en finalisant » (« Resume and finalize » en anglais). Un clic sur « Reprendre » SHALL demander au backend la reprise de ce seul worker, sans arrêter le pool ni interrompre les autres workers. Un clic sur « Reprendre en finalisant » SHALL demander la même reprise avec `finalize_only`. Le bouton « Reprendre en finalisant » SHALL être désactivé tant que le `tasks.md` du worktree du change n'est pas entièrement coché, et SHALL alors indiquer (texte ou info-bulle) le nombre de tâches restantes. Pendant une requête, les deux boutons du worker SHALL être désactivés pour éviter une double demande. Si la reprise échoue, le panneau SHALL afficher l'erreur retournée par le backend et conserver le worker en pause. Les boutons SHALL NOT apparaître pour les workers dans un autre statut.

#### Scenario: Bouton visible sur un worker en pause
- **WHEN** l'utilisateur ouvre le panneau d'état alors que le worker 1, sur le changement `add-user-auth`, est `paused`
- **THEN** la ligne de ce worker affiche sa raison de blocage et les boutons « Reprendre » et « Reprendre en finalisant »

#### Scenario: Reprise d'un worker
- **WHEN** l'utilisateur clique sur « Reprendre » pour le worker en pause du changement `add-user-auth`
- **THEN** l'application envoie la demande de reprise pour ce worker, le panneau reste ouvert, et la ligne reflète le nouveau statut du worker dès sa reprise par le pool, sans rechargement de la page

#### Scenario: Reprise en finalisant disponible
- **WHEN** le change du worker en pause a toutes ses tâches cochées dans son worktree et que l'utilisateur clique sur « Reprendre en finalisant »
- **THEN** l'application envoie la demande de reprise avec `finalize_only` pour ce worker et la ligne reflète le nouveau statut dès sa reprise par le pool

#### Scenario: Reprise en finalisant indisponible
- **WHEN** le change du worker en pause a encore 1 tâche non cochée dans son worktree
- **THEN** le bouton « Reprendre en finalisant » est désactivé et indique qu'il reste 1 tâche, tandis que « Reprendre » reste actif

#### Scenario: Aucun bouton hors pause
- **WHEN** un worker est au statut `working`, `testing` ou `healing`
- **THEN** sa ligne n'affiche aucun bouton de reprise

#### Scenario: Échec de la reprise
- **WHEN** le backend refuse la reprise, par exemple parce que le worker n'est plus en pause
- **THEN** le panneau affiche le message d'erreur et les boutons redeviennent actifs
