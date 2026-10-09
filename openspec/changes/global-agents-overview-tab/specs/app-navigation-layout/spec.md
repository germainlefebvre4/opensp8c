## MODIFIED Requirements

### Requirement: Barre principale globale
L'application SHALL afficher en permanence, tout en haut de la fenêtre et sur toute sa largeur, une barre principale globale contenant la marque « OpenSpec », les entrées « Projects », « Agents » et « Configuration » (dans cet ordre), et le dropdown de sélection de l'agent CLI actif. Le dropdown de l'agent SHALL être aligné à l'extrémité droite de la barre. Cette barre SHALL rester affichée quelle que soit la route courante, y compris lorsqu'aucun workspace n'est configuré.

#### Scenario: Contenu de la barre principale
- **WHEN** l'utilisateur consulte n'importe quelle page de l'application
- **THEN** la barre principale est affichée en haut sur toute la largeur de la fenêtre
- **THEN** elle contient la marque « OpenSpec », les entrées « Projects », « Agents » et « Configuration » et le dropdown de l'agent actif à droite

#### Scenario: Entrée active mise en évidence
- **WHEN** l'utilisateur est sur une page de workspace (Kanban, Specs, Timeline, Agents ou Settings)
- **THEN** l'entrée « Projects » est visuellement marquée comme active, ni « Agents » ni « Configuration » ne le sont

#### Scenario: Barre présente sans workspace
- **WHEN** aucun workspace n'est configuré
- **THEN** la barre principale reste affichée avec ses entrées et le dropdown de l'agent

### Requirement: Retour aux projets depuis Configuration
Lorsque l'utilisateur active l'entrée « Projects » depuis une page globale (Configuration ou Agents), l'application SHALL le ramener à la dernière page de workspace qu'il avait consultée (onglet et workspace) durant la session. Les pages globales SHALL NE PAS être mémorisées comme dernière page de workspace. S'il n'en existe aucune, l'application SHALL afficher le Kanban du workspace par défaut.

#### Scenario: Retour à la dernière page consultée
- **WHEN** l'utilisateur consulte l'onglet Timeline du workspace `B`, ouvre Configuration puis clique sur « Projects »
- **THEN** l'application affiche l'onglet Timeline du workspace `B`

#### Scenario: Retour depuis la vue Agents globale
- **WHEN** l'utilisateur consulte l'onglet Timeline du workspace `B`, ouvre « Agents » dans la barre principale puis clique sur « Projects »
- **THEN** l'application affiche l'onglet Timeline du workspace `B`

#### Scenario: Aucun historique de navigation
- **WHEN** l'application est chargée directement sur Configuration ou sur la vue Agents globale puis l'utilisateur clique sur « Projects »
- **THEN** l'application affiche le Kanban du premier workspace configuré

#### Scenario: Workspace supprimé entre-temps
- **WHEN** le workspace mémorisé n'existe plus au moment du retour vers « Projects »
- **THEN** l'application affiche le Kanban du premier workspace restant, ou l'écran d'accueil si la liste est vide

### Requirement: Configuration en pleine largeur
Sur les pages globales Configuration et Agents (`/agents-overview`), l'application SHALL masquer la sidebar des projets et le sous-menu du workspace, et la page SHALL occuper toute la largeur de la fenêtre sous la barre principale. La barre principale SHALL rester affichée avec l'entrée correspondant à la page courante mise en évidence. L'application SHALL NE PAS ajouter le paramètre `workspace` à l'URL de ces pages.

#### Scenario: Sidebar et sous-menu masqués
- **WHEN** l'utilisateur ouvre Configuration ou la vue Agents globale
- **THEN** ni la sidebar des projets ni le sous-menu du workspace ne sont affichés
- **THEN** la page occupe toute la largeur sous la barre principale

#### Scenario: Entrée Configuration active
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'entrée « Configuration » de la barre principale est marquée comme active et « Projects » et « Agents » ne le sont pas

#### Scenario: Entrée Agents active
- **WHEN** l'utilisateur ouvre la vue Agents globale
- **THEN** l'entrée « Agents » de la barre principale est marquée comme active et « Projects » et « Configuration » ne le sont pas

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré
- **THEN** les entrées « Agents » et « Configuration » restent visibles et leur activation affiche la page correspondante
