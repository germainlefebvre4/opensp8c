# Proposal

## Why

Dans un workspace, les pages Specs et Timeline partagent une barre de sous-onglets compacte, sans titre de page, mais Agents et Settings sont agencées autrement : Agents n'a aucun sous-onglet (workers et runs récents sont empilés sur une seule page qui défile) et Settings affiche un `<h1>` puis des sous-onglets dans un second style. La navigation interne d'un workspace est donc incohérente d'une page à l'autre, et le style des sous-onglets est dupliqué quatre fois dans le code.

## What Changes

- Agents : la page est découpée en deux sous-onglets, **Workers** (résumé du pool + table des workers) et **Runs** (runs récents). Le panneau de détail de run reste à droite et reste utilisable depuis les deux sous-onglets.
- Agents : le sous-onglet actif est porté par l'URL (`?tab=runs`, `?tab=workers`). Workers est le défaut, sans paramètre. Un lien `?run=` sans `tab` explicite ouvre le sous-onglet Runs.
- Settings : l'en-tête `<h1>` disparaît. La barre de sous-onglets adopte le style de Specs/Timeline. Le nom du workspace reste affiché, à l'extrémité droite de la barre. Le sous-onglet actif reste dans l'URL (`?tab=`), sans changement de comportement.
- Agents n'a plus d'en-tête `<h1>` : la barre de sous-onglets occupe le haut de la page, comme Specs et Timeline.
- Un composant partagé `SubTabs` remplace les implémentations inline de SpecsPage, TimelinePage et SettingsPage, sans changement visuel ni fonctionnel pour Specs et Timeline.
- Hors périmètre : ConfigurationPage (page globale, hors workspace) garde son style actuel. Le style de `WorkspaceTabs` (onglets principaux) ne change pas.

## Capabilities

### New Capabilities
- `workspace-page-subtabs`: barre de sous-onglets commune aux pages d'un workspace (Specs, Timeline, Agents, Settings) : emplacement, absence de titre de page, style, emplacement optionnel de fin de barre.

### Modified Capabilities
- `workspace-agent-pool-tab`: la page Agents est organisée en sous-onglets Workers et Runs, avec l'onglet actif dans l'URL ; la disponibilité de la section des runs récents dans le cas « aucun pool actif » passe par le sous-onglet Runs.
- `agent-run-detail`: la section « Runs récents » vit dans le sous-onglet Runs et non plus sous la liste des workers ; le panneau de détail s'ouvre depuis les deux sous-onglets et un lien `?run=` ouvre Runs.
- `workspace-settings`: les sous-onglets de Settings adoptent la barre commune, sans titre de page, avec le nom du workspace en fin de barre et l'onglet actif dans l'URL.

## Impact

- Frontend uniquement : `frontend/src/pages/AgentsPage.tsx`, `SettingsPage.tsx`, `SpecsPage.tsx`, `TimelinePage.tsx`, nouveau `frontend/src/components/SubTabs.tsx`, locales `agents` (fr/en) et `settings` si des libellés changent.
- Tests à adapter : `AgentsPage.test.tsx`, `AgentsPage.interaction.test.tsx` (les runs récents ne sont plus visibles par défaut), `SettingsPage.test.tsx` (plus de titre `<h1>`).
- Aucun changement backend ni d'API.
- Coordination : `TimelinePage.tsx` est aussi touché par le change en cours `improve-matrix-change-drilldown-nav` ; ici seule la barre de sous-onglets est remplacée, le reste du fichier n'est pas modifié.
