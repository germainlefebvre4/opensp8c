# Design

## Context

Le crash vient d'une chaîne de trois maillons : un YAML `tags` sans `type`, `Tags.Type []string` qui reste nil côté Go et se sérialise en `"type": null`, puis `c.tags?.type.map(...)` dans `TimelineChangeCard.tsx` (ligne 43) qui lève un `TypeError`. `loadChange` (`backend/internal/openspec/change.go`) normalise déjà `AgentSpecialization` nil en `[]string{}` mais pas `Type`. Les autres consommateurs du frontend (`ChangeCard`, `DetailPanel`, `TimelinePage`, `KanbanPage`) utilisent déjà un accès défensif ; `TimelineChangeCard` est le seul à ne pas le faire. Voir proposal.md pour la motivation.

## Goals / Non-Goals

**Goals:**
- Supprimer le crash à la source (contrat API respecté) et au point de consommation (composant tolérant).
- Couvrir les deux côtés par un test de régression.

**Non-Goals:**
- Ajouter un `ErrorBoundary` global : changement transverse, à traiter dans son propre change.
- Corriger `'○'.repeat(5 - complexity)` (RangeError si `complexity > 5`) : cas distinct, hors périmètre.
- Migrer ou réécrire les `.openspec.yaml` existants.

## Decisions

1. **Défense en profondeur : correction backend et frontend.**
   - Backend : dans `loadChange`, à côté de la normalisation d'`AgentSpecialization`, remplacer `Tags.Type` nil par `[]string{}`. Cela rend vrai le contrat déjà décrit (`type` est un tableau).
   - Frontend : `c.tags?.type?.map(...)` dans `TimelineChangeCard`, aligné sur `ChangeCard`.
   - Alternative écartée : ne corriger qu'un seul côté. Backend seul laisse le composant fragile face à un autre producteur ou à un cache ancien ; frontend seul laisse une API qui viole son contrat pour les autres clients.

2. **Normalisation dans `loadChange` plutôt que dans le type `Tags`.**
   `loadChange` est le point où `AgentSpecialization` est déjà normalisé ; on suit ce modèle. Alternative écartée : un `omitempty` ou un `MarshalJSON` custom sur `Tags`, plus invasif et qui changerait la forme de la réponse (champ absent au lieu de `[]`).

3. **Test frontend : `renderToStaticMarkup` dans un `MemoryRouter`, avec i18n initialisé.**
   `TimelineChangeCard` utilise `Link`, `useSearchParams` et `useTranslation('timeline')` ; le test suit le modèle de `WorkspaceTabs.test.tsx` (Vitest, `renderToStaticMarkup`, `MemoryRouter`, i18next initialisé avec `locales/en/timeline.json`). Sans correctif, le rendu avec `tags` sans `type` lève l'exception, ce qui prouve la régression.

## Risks / Trade-offs

- [Un autre champ de `Tags` peut aussi être nil] -> `components` est déjà lu avec `?? []` côté page et `agent_specialization` est normalisé ; on ne traite que `type`, seul champ observé en cause.
- [Le `type` vide masque un problème de tagging] -> l'absence de badge est le comportement voulu par la spec `change-tags` (`type` peut être vide) ; le retag manuel reste disponible.
