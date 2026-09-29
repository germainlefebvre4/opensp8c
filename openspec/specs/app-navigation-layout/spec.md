# app-navigation-layout Specification

## Purpose

Définit la structure de navigation de l'application : une barre principale globale, une sidebar listant les projets, un sous-menu d'onglets propre au workspace sélectionné, et le comportement de la page Configuration qui s'affiche en pleine largeur.

## Requirements

### Requirement: Barre principale globale
L'application SHALL afficher en permanence, tout en haut de la fenêtre et sur toute sa largeur, une barre principale globale contenant la marque « OpenSpec », les entrées « Projects » et « Configuration », et le dropdown de sélection de l'agent CLI actif. Le dropdown de l'agent SHALL être aligné à l'extrémité droite de la barre. Cette barre SHALL rester affichée quelle que soit la route courante, y compris lorsqu'aucun workspace n'est configuré.

#### Scenario: Contenu de la barre principale
- **WHEN** l'utilisateur consulte n'importe quelle page de l'application
- **THEN** la barre principale est affichée en haut sur toute la largeur de la fenêtre
- **THEN** elle contient la marque « OpenSpec », les entrées « Projects » et « Configuration » et le dropdown de l'agent actif à droite

#### Scenario: Entrée active mise en évidence
- **WHEN** l'utilisateur est sur une page de workspace (Kanban, Specs, Timeline, Agents ou Settings)
- **THEN** l'entrée « Projects » est visuellement marquée comme active et « Configuration » ne l'est pas

#### Scenario: Barre présente sans workspace
- **WHEN** aucun workspace n'est configuré
- **THEN** la barre principale reste affichée avec ses entrées et le dropdown de l'agent

### Requirement: Sidebar des projets sous la barre principale
L'application SHALL afficher, sur les pages de workspace, une sidebar verticale à gauche, située sous la barre principale, listant tous les projets avec leurs compteurs, et proposant l'ajout et la suppression d'un projet ainsi que le repli de la sidebar. La sidebar SHALL NE PAS contenir l'entrée Configuration ni le sélecteur d'agent, qui appartiennent à la barre principale.

#### Scenario: Sidebar limitée aux projets
- **WHEN** l'utilisateur consulte une page de workspace
- **THEN** la sidebar affiche la liste des projets, le bouton d'ajout et le bouton de repli
- **THEN** elle n'affiche ni entrée Configuration ni sélecteur d'agent

#### Scenario: Sidebar positionnée sous la barre principale
- **WHEN** l'utilisateur consulte une page de workspace
- **THEN** la sidebar commence sous la barre principale et n'occupe pas la hauteur de celle-ci

### Requirement: Sous-menu du workspace
L'application SHALL afficher, sur les pages de workspace, un sous-menu horizontal en haut de la zone de contenu du workspace, sous la barre principale, entre la sidebar et le bord droit de la fenêtre. Ce sous-menu SHALL lister les onglets Kanban, Specs, Timeline, Agents et Settings, mettre en évidence l'onglet actif et conserver le paramètre `workspace` de l'URL lors de la navigation entre onglets.

#### Scenario: Onglets du sous-menu
- **WHEN** l'utilisateur consulte une page de workspace
- **THEN** le sous-menu affiche, dans cet ordre, Kanban, Specs, Timeline, Agents et Settings
- **THEN** il est situé dans la zone à droite de la sidebar et non sur toute la largeur de la fenêtre

#### Scenario: Navigation entre onglets
- **WHEN** l'utilisateur clique sur l'onglet Specs alors que `?workspace=<id>` est dans l'URL
- **THEN** l'URL résultante est `/specs?workspace=<id>` et l'onglet Specs est mis en évidence

#### Scenario: Sous-menu affiché sans workspace
- **WHEN** aucun workspace n'est configuré
- **THEN** le sous-menu reste affiché au-dessus de la zone de contenu, qui présente l'invitation à ajouter un premier projet

### Requirement: Retour aux projets depuis Configuration
Lorsque l'utilisateur active l'entrée « Projects » depuis Configuration, l'application SHALL le ramener à la dernière page de workspace qu'il avait consultée (onglet et workspace) durant la session. S'il n'en existe aucune, l'application SHALL afficher le Kanban du workspace par défaut.

#### Scenario: Retour à la dernière page consultée
- **WHEN** l'utilisateur consulte l'onglet Timeline du workspace `B`, ouvre Configuration puis clique sur « Projects »
- **THEN** l'application affiche l'onglet Timeline du workspace `B`

#### Scenario: Aucun historique de navigation
- **WHEN** l'application est chargée directement sur Configuration puis l'utilisateur clique sur « Projects »
- **THEN** l'application affiche le Kanban du premier workspace configuré

#### Scenario: Workspace supprimé entre-temps
- **WHEN** le workspace mémorisé n'existe plus au moment du retour vers « Projects »
- **THEN** l'application affiche le Kanban du premier workspace restant, ou l'écran d'accueil si la liste est vide

### Requirement: Configuration en pleine largeur
Sur la page Configuration, l'application SHALL masquer la sidebar des projets et le sous-menu du workspace, et la page Configuration SHALL occuper toute la largeur de la fenêtre sous la barre principale. La barre principale SHALL rester affichée avec l'entrée « Configuration » mise en évidence.

#### Scenario: Sidebar et sous-menu masqués
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** ni la sidebar des projets ni le sous-menu du workspace ne sont affichés
- **THEN** la page Configuration occupe toute la largeur sous la barre principale

#### Scenario: Entrée Configuration active
- **WHEN** l'utilisateur ouvre Configuration
- **THEN** l'entrée « Configuration » de la barre principale est marquée comme active et « Projects » ne l'est pas

#### Scenario: Accès sans workspace configuré
- **WHEN** aucun workspace n'est configuré
- **THEN** l'entrée « Configuration » reste visible et son activation affiche la page Configuration
