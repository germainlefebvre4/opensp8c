# Spec Delta

## MODIFIED Requirements

### Requirement: Onglet Agents interactif avec panneau de détail
L'onglet Agents d'un workspace SHALL permettre de sélectionner par clic une ligne de worker actif (sous-onglet « Workers ») ou de run récent (sous-onglet « Runs »). La sélection SHALL ouvrir un panneau de détail latéral, à côté de la liste, qui reste visible pendant la consultation du détail afin de changer de sélection d'un clic, et qui reste ouvert lorsque l'utilisateur change de sous-onglet. Le panneau SHALL afficher : le change, le worker, l'issue du run avec sa raison de blocage le cas échéant, la durée, une frise chronologique des entrées et la liste chronologique de ces entrées (narration de l'agent, appels d'outils avec durée, transitions de statut, commits). Un appel d'outil SHALL pouvoir être déplié pour afficher son entrée et son résultat. La sélection SHALL pouvoir être annulée pour fermer le panneau.

#### Scenario: Sélection d'un worker actif
- **WHEN** l'utilisateur clique sur la ligne du worker qui traite `add-user-auth`
- **THEN** le panneau de détail s'ouvre sur le run en cours de ce worker

#### Scenario: Sélection d'un run récent
- **WHEN** l'utilisateur clique sur un run terminé dans la section des runs récents du sous-onglet « Runs »
- **THEN** le panneau de détail affiche ce run avec son issue finale et sa date de fin

#### Scenario: Changement de sélection
- **WHEN** un panneau est ouvert sur un run et que l'utilisateur clique sur une autre ligne
- **THEN** le panneau affiche le run de la nouvelle sélection sans fermer ni recharger la liste

#### Scenario: Panneau conservé au changement de sous-onglet
- **WHEN** un panneau est ouvert sur un run et que l'utilisateur bascule entre les sous-onglets « Workers » et « Runs »
- **THEN** le panneau reste ouvert sur le même run

#### Scenario: Dépliage d'un appel d'outil
- **WHEN** l'utilisateur déplie un appel d'outil dans la liste d'activité
- **THEN** l'entrée de l'outil et son résultat (éventuellement tronqué) sont affichés

### Requirement: Section des runs récents
Le sous-onglet « Runs » de l'onglet Agents SHALL afficher une section « Runs récents » listant les runs pool du workspace, y compris ceux dont le worker n'existe plus. Chaque ligne SHALL indiquer le change, la date de début, la durée, l'issue et, pour une issue `paused`, la raison de blocage. Lorsque aucun run n'existe, la section SHALL afficher un état vide.

#### Scenario: Run terminé visible après la disparition du worker
- **WHEN** un worker termine le change `fix-docs` et disparaît de la liste des workers actifs
- **THEN** la section des runs récents du sous-onglet « Runs » affiche ce run avec son issue

#### Scenario: Aucun run
- **WHEN** aucun run pool n'existe pour le workspace
- **THEN** la section des runs récents affiche un état vide
