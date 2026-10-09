# Spec Delta

## MODIFIED Requirements

### Requirement: Sidebar des projets sous la barre principale
L'application SHALL afficher, sur les pages de workspace, une sidebar verticale à gauche, située sous la barre principale, listant tous les projets avec leur nombre de changes en attente d'action et la répartition de leurs changes par statut (barre segmentée), et proposant l'ajout et la suppression d'un projet ainsi que le repli de la sidebar. Chaque projet SHALL occuper une ligne de nom (ligne 1) et une ligne de répartition (ligne 2) ; la ligne entière SHALL être cliquable pour sélectionner le projet. La sidebar SHALL NE PAS contenir l'entrée Configuration ni le sélecteur d'agent, qui appartiennent à la barre principale.

#### Scenario: Sidebar limitée aux projets
- **WHEN** l'utilisateur consulte une page de workspace
- **THEN** la sidebar affiche la liste des projets, le bouton d'ajout et le bouton de repli
- **THEN** elle n'affiche ni entrée Configuration ni sélecteur d'agent

#### Scenario: Sidebar positionnée sous la barre principale
- **WHEN** l'utilisateur consulte une page de workspace
- **THEN** la sidebar commence sous la barre principale et n'occupe pas la hauteur de celle-ci

#### Scenario: Projet sur deux lignes
- **WHEN** la sidebar est ouverte
- **THEN** chaque projet affiche son nom et son compteur d'actions en attente sur la ligne 1, et la barre segmentée avec le total de ses changes sur la ligne 2

#### Scenario: Ligne projet entièrement cliquable
- **WHEN** l'utilisateur clique n'importe où sur la ligne d'un projet, hors contrôles propres (chevron, menu d'actions)
- **THEN** ce projet devient le workspace actif
