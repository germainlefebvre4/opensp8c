## MODIFIED Requirements

### Requirement: Mode Matrice dans la Timeline
La TimelinePage SHALL proposer deux sous-onglets « Changes » et « Matrice » en haut de page, à la place d'un en-tête de page avec titre. Le sous-onglet « Matrice » affiche une grille spec × bucket temporel représentant l'intensité d'activité de chaque spec dans le temps, ainsi qu'un panel droit de détail spec ouvert au clic. Le sous-onglet actif SHALL être conservé uniquement dans l'état local de la page (non reflété dans l'URL).

#### Scenario: Activation du sous-onglet Matrice
- **WHEN** l'utilisateur clique sur le sous-onglet "Matrice" de la TimelinePage
- **THEN** la grille spec × bucket temporel s'affiche, chaque ligne correspondant à une spec, chaque colonne à un bucket temporel (jour, semaine, mois ou trimestre selon la granularité sélectionnée) de la période couverte par les changes, et le sous-onglet "Matrice" est visuellement marqué actif

#### Scenario: Sous-onglet Changes actif par défaut
- **WHEN** l'utilisateur navigue vers `/timeline`
- **THEN** le sous-onglet Changes est actif par défaut

#### Scenario: Pas de titre de page
- **WHEN** la TimelinePage est affichée
- **THEN** la barre de sous-onglets occupe le haut de la page et aucun titre "Timeline" ni toggle en pilule n'est affiché

#### Scenario: Deep-link vers Matrice avec spec pré-sélectionnée
- **WHEN** l'URL contient le paramètre `?spec=<name>`
- **THEN** le sous-onglet Matrice s'active et le panel droit s'ouvre directement sur la spec nommée
