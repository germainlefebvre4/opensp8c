## ADDED Requirements

### Requirement: Capacité du pool visible sur le bouton d'en-tête
Lorsqu'un pool est actif pour le workspace affiché, le bouton d'en-tête du Kanban SHALL indiquer la capacité du pool sous la forme « actifs/taille », où « actifs » est le nombre de workers actuellement assignés à un change (y compris les workers en pause) et « taille » la taille configurée du pool. Cette indication SHALL être visible sans ouvrir le panneau d'état, y compris lorsque aucun worker n'est encore assigné (« 0/taille »). Lorsqu'aucun pool n'est actif, le bouton SHALL n'afficher aucune capacité. Cette indication SHALL rester à jour tant que la page est ouverte, sans rechargement manuel.

#### Scenario: Pool qui vient de démarrer sans worker assigné
- **WHEN** l'utilisateur démarre un pool de taille 3 sur un workspace et qu'aucun worker n'est encore assigné à un change
- **THEN** le bouton d'en-tête affiche l'état « pool actif » accompagné de la capacité « 0/3 »

#### Scenario: Capacité mise à jour lorsqu'un worker est assigné
- **WHEN** un worker est assigné à un change sur un pool actif de taille 3 alors que l'utilisateur consulte le Kanban
- **THEN** le bouton d'en-tête affiche « 1/3 » sans que l'utilisateur ait à recharger la page

#### Scenario: Aucun pool actif
- **WHEN** aucun pool n'est actif sur le workspace affiché
- **THEN** le bouton d'en-tête affiche « Lancer le Pool » sans indication de capacité
