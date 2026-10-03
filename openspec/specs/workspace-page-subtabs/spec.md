# workspace-page-subtabs Specification

## Purpose
Donne aux pages d'un workspace qui ont des sous-onglets (Specs, Timeline, Agents, Settings) une barre de sous-onglets unique, au même emplacement et au même style, sans titre de page, pour que la navigation interne d'un workspace soit cohérente d'une page à l'autre.

## Requirements

### Requirement: Barre de sous-onglets commune aux pages du workspace
Les pages Specs, Timeline, Agents et Settings d'un workspace SHALL afficher leurs sous-onglets dans une barre unique placée tout en haut de la zone de contenu, sous les onglets principaux du workspace, avec le même style sur ces quatre pages. Aucune de ces pages ne SHALL afficher de titre de page (`<h1>`) au-dessus de cette barre. L'onglet actif SHALL être visuellement distinct des autres.

#### Scenario: Barre en haut de page, sans titre
- **WHEN** l'utilisateur ouvre l'une des pages Specs, Timeline, Agents ou Settings d'un workspace
- **THEN** la barre de sous-onglets occupe le haut de la zone de contenu et aucun titre de page n'est affiché au-dessus ou à côté de ses onglets

#### Scenario: Même style d'une page à l'autre
- **WHEN** l'utilisateur passe de Specs à Timeline, Agents puis Settings
- **THEN** la hauteur, l'espacement, la couleur et le marquage de l'onglet actif de la barre de sous-onglets sont identiques sur les quatre pages

#### Scenario: Onglet actif marqué
- **WHEN** un sous-onglet est actif sur l'une de ces pages
- **THEN** il est marqué comme actif de façon visuellement distincte, et les autres sous-onglets sont affichés en style inactif

### Requirement: Information complémentaire en fin de barre
La barre de sous-onglets SHALL pouvoir afficher une information complémentaire alignée à son extrémité droite, séparée des onglets, sans qu'elle soit un onglet ni qu'elle réagisse au clic comme un onglet.

#### Scenario: Information affichée à droite
- **WHEN** une page fournit une information complémentaire pour sa barre de sous-onglets
- **THEN** cette information est affichée à l'extrémité droite de la barre, sur la même ligne que les onglets

#### Scenario: Pas d'information complémentaire
- **WHEN** une page n'en fournit pas
- **THEN** la barre n'affiche que ses onglets, sans zone vide interactive
