# Proposal

## Why

La navigation mélange aujourd'hui des éléments globaux (Configuration, sélecteur d'agent CLI, liste des projets) dans la sidebar et des onglets propres à un workspace dans une barre du haut qui reste affichée même sur la page Configuration, où ils n'ont pas de sens. Séparer clairement ce qui est global (barre principale, liste des projets) de ce qui est propre au workspace sélectionné (sous-menu d'onglets) rend l'écran plus lisible et prépare l'arrivée des réglages par workspace qui surchargeront Configuration.

## What Changes

- Ajout d'une **barre principale globale** en haut, sur toute la largeur de la fenêtre : marque « OpenSpec », entrées « Projects » et « Configuration », et le dropdown de sélection de l'agent CLI actif aligné à droite.
- La **sidebar gauche** reste un menu vertical listant tous les projets (badges, suppression, ajout, repli), mais perd l'entrée Configuration et le sélecteur d'agent, déplacés dans la barre principale. Elle démarre désormais sous la barre principale.
- Les onglets du workspace (Kanban, Specs, Timeline, Agents, Settings) forment un **sous-menu** placé en haut de la zone workspace, entre la sidebar et le bord droit de la fenêtre, sous la barre principale. « Settings » reste un onglet du workspace (sa surcharge de Configuration par workspace est prévue dans un change ultérieur ; son contenu n'est pas modifié ici).
- Sur **Configuration**, la sidebar et le sous-menu disparaissent : la page occupe toute la largeur sous la barre principale. **BREAKING** : l'exigence « Sidebar conservée sur Configuration » est supprimée, ainsi que l'icône Configuration en panneau réduit (l'entrée est désormais toujours dans la barre principale).
- Un clic sur « Projects » depuis Configuration ramène à la dernière page de workspace consultée (onglet et workspace), au lieu de retomber sur le premier workspace.
- Le dropdown de l'agent reçoit une largeur minimale pour rester lisible dans une barre étroite et s'aligne sur le bord droit de son bouton.
- Libellé UI « Projects » ; le vocabulaire technique (« workspace », routes, query param) est inchangé.

## Capabilities

### New Capabilities
- `app-navigation-layout`: structure de navigation de l'application : barre principale globale, sidebar des projets, sous-menu d'onglets du workspace, comportement sur la page Configuration et retour vers les projets.

### Modified Capabilities
- `platform-configuration`: l'entrée Configuration passe dans la barre principale ; la page s'affiche sans sidebar ni sous-menu ; suppression de l'accessibilité en panneau réduit.
- `agent-selector-dropdown`: le sélecteur est dans la barre principale ; le dropdown s'aligne sur le bord droit du bouton avec une largeur minimale.
- `workspace-agent-pool-tab`: l'onglet Agents appartient au sous-menu du workspace et non plus à la « navigation principale ».
- `agent-specialization`: l'écran Settings est un onglet du sous-menu du workspace.

## Impact

- Frontend : `components/Layout.tsx` (restructuration, nouvelle barre principale et sous-menu, mémorisation de la dernière route projets), `components/WorkspaceSidebar.tsx` (retrait de Configuration et `AgentSelector`, adaptation de `WorkspaceSidebar.test.tsx`), `components/AgentSelector.tsx` (largeur minimale et alignement), `locales/{en,fr}/navigation.json` (clé « projects »).
- Aucun changement backend, API ou routes ; le paramètre `?workspace=` est conservé.
- Specs existantes à mettre à jour via des deltas (voir ci-dessus).
