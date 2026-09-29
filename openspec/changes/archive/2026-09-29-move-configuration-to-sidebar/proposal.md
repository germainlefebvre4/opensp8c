# Proposal

## Why

Configuration vit aujourd'hui comme une pill isolée dans la nav du haut, à côté des onglets workspace (Kanban/Specs/Timeline/Agents/Réglages) qu'elle n'est pourtant pas censée rejoindre conceptuellement. Le panneau de gauche (`WorkspaceSidebar`), lui, ne contient que du contenu workspace (agent par défaut, liste des projets) alors qu'il est l'endroit naturel pour regrouper les fonctionnalités globales, hors contexte d'un workspace précis. Regrouper Configuration et la gestion de la langue dans ce panneau clarifie la séparation : nav du haut = onglets d'un workspace actif, panneau de gauche = fonctionnalités globales.

## What Changes

- Retirer l'entrée `Configuration` de la nav du haut (`Layout.tsx`) et l'ajouter en haut du panneau de gauche (`WorkspaceSidebar`), au-dessus du bloc "Projets" (agent par défaut + liste des workspaces).
- L'entrée `Configuration` du panneau de gauche reste cliquable même quand le panneau est réduit (`w-8`) : une icône seule reste visible et active, alors que le reste du contenu du panneau (agent par défaut, projets) reste masqué en mode réduit.
- Retirer le `LanguageSwitcher` de la nav du haut.
- Ajouter un nouveau sous-onglet "Langue" dans `ConfigurationPage` (aux côtés de "Agents" et "CLI"), qui héberge le composant `LanguageSwitcher` existant.
- **BREAKING** (comportement utilisateur) : la langue et la Configuration ne sont plus accessibles depuis la nav du haut ; elles déménagent dans le panneau de gauche / Configuration > Langue.

## Capabilities

### New Capabilities
(aucune)

### Modified Capabilities

- `platform-configuration`: l'entrée de navigation Configuration est désormais dans le panneau de gauche (et non plus dans la nav du haut distincte des onglets workspace) ; elle reste accessible via une icône même quand le panneau de gauche est réduit ; nouveau sous-onglet "Langue" dans Configuration.
- `i18n-core`: le sélecteur de langue n'est plus affiché dans la nav du haut ; il est affiché dans Configuration > Langue.

## Impact

- Frontend uniquement : `frontend/src/components/Layout.tsx`, `frontend/src/components/WorkspaceSidebar.tsx`, `frontend/src/pages/ConfigurationPage.tsx`, `frontend/src/components/LanguageSwitcher.tsx` (réutilisé tel quel), locales `navigation.json` et `configuration.json` (fr/en).
- Pas de changement d'API backend, pas de changement de schéma de données.
- Tests à ajuster : tout test qui cherche le lien Configuration ou le LanguageSwitcher dans la nav du haut (`Layout` / navigation tests) et `ConfigurationPage.test.tsx` pour le nouvel onglet.
