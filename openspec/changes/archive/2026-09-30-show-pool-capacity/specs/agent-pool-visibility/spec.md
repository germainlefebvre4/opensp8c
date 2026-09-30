## ADDED Requirements

### Requirement: Capacité du pool et workers disponibles visibles sans worker assigné
La vue Agent Pool, dans la section Configuration > Agent Pool comme dans l'onglet Agents d'un workspace, SHALL afficher pour tout pool actif sa capacité sous la forme « X/Y workers actifs » ainsi que son mode de délégation, y compris lorsqu'aucun worker n'est assigné à un change. Lorsqu'un pool est actif sans aucun worker assigné, la vue SHALL indiquer le nombre de workers disponibles (égal à la taille du pool) et qu'ils attendent un change prêt, au lieu de l'état vide « aucun agent actif ». Lorsqu'un pool est actif avec des workers assignés mais moins que sa taille, la vue SHALL indiquer sous le tableau le nombre de workers restant disponibles. L'état vide « aucun agent actif » SHALL être réservé au cas où aucun pool ne tourne (pour l'onglet Agents : aucun pool sur le workspace affiché). Le nombre de workers disponibles SHALL être calculé comme la taille du pool moins le nombre de workers assignés.

#### Scenario: Pool actif sans worker assigné dans l'onglet Agents
- **WHEN** un pool de taille 3 vient d'être démarré sur le workspace `A` et aucun worker n'est encore assigné, et l'utilisateur ouvre l'onglet Agents de `A`
- **THEN** la vue affiche « 0/3 workers actifs », le mode de délégation, et « 3 workers disponibles, en attente d'un change prêt », et n'affiche pas « aucun agent actif »

#### Scenario: Pool actif sans worker assigné dans Configuration
- **WHEN** un pool de taille 3 est actif sur le workspace `A` sans worker assigné et l'utilisateur ouvre Configuration > Agent Pool
- **THEN** la section du workspace `A` affiche « 0/3 workers actifs », le mode de délégation et « 3 workers disponibles » à la place d'un tableau vide

#### Scenario: Pool partiellement occupé
- **WHEN** un pool de taille 3 a 2 workers assignés à des changes
- **THEN** la vue affiche « 2/3 workers actifs », le tableau des 2 workers, et « 1 worker disponible » sous le tableau

#### Scenario: Aucun pool actif sur le workspace
- **WHEN** aucun pool ne tourne sur le workspace affiché dans l'onglet Agents
- **THEN** la vue affiche l'état vide « aucun agent actif » sans indication de workers disponibles

#### Scenario: Mise à jour du compteur
- **WHEN** un worker est assigné à un change ou se libère pendant que l'utilisateur consulte la vue
- **THEN** les compteurs « X/Y workers actifs » et « N workers disponibles » reflètent ce changement sans rechargement manuel
