## Why

Le DetailPanel et l'ExplorePanel s'affichent en overlay `position: fixed` par-dessus les colonnes Kanban, rendant le board illisible lorsqu'un panel est ouvert. Par ailleurs, les onglets Proposal et Design du DetailPanel n'affichent que du texte brut (`<pre>`), rendant la lecture des artifacts inconfortable.

## What Changes

- DetailPanel passe de `position: fixed` (overlay) à un slot inline à droite des colonnes Kanban
- ExplorePanel applique le même traitement : slot inline à droite, largeur fixe
- KanbanPage adopte un layout flex horizontal : `[colonnes flex:1] [panel 400px]`
- Les colonnes sont scrollables horizontalement si nécessaire quand le panel est ouvert
- Onglets Proposal et Design du DetailPanel : ajout d'un toggle Raw / Rendu (ReactMarkdown) partagé entre les deux onglets

## Capabilities

### New Capabilities

_(aucune : les changements sont des modifications comportementales des capabilities existantes)_

### Modified Capabilities

- `kanban-board` : le layout doit partager l'espace horizontal avec le panel quand il est ouvert
- `kanban-change-detail` : le DetailPanel et l'ExplorePanel deviennent des éléments inline ; les artifacts proposal/design peuvent être rendus en Markdown

## Impact

- `frontend/src/pages/KanbanPage.tsx` : layout flex horizontal, slot conditionnel pour le panel
- `frontend/src/components/DetailPanel.tsx` : suppression `position: fixed`, ajout toggle Raw/Rendu
- `frontend/src/components/ExplorePanel.tsx` : suppression `position: fixed`
- Aucune modification backend ni d'API
