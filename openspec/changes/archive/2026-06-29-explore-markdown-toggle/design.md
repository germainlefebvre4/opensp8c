## Context

Les panels d'exploration (`ExploreAnonymousPanel`, `ExplorePanel`) affichent les messages IA en texte brut via `whitespace-pre-wrap`. Le projet utilise déjà `react-markdown` (v10) dans `DetailPanel` avec un toggle `raw/rendered` identique à ce qui est demandé. La préférence utilisateur doit survivre au rechargement de page via `localStorage`.

## Goals / Non-Goals

**Goals:**
- Toggle global `raw/rendered` dans le header des deux panels d'exploration
- Persistance de la préférence via `localStorage` (clé `explore-view-mode`, défaut `raw`)
- Rendu `ReactMarkdown` pour les messages assistant en mode `rendered`
- Messages utilisateur toujours en raw (texte simple)
- Messages partiels (streaming) rendus même en mode `rendered`

**Non-Goals:**
- Toggle par-message
- Persistance côté serveur
- Modification du `ExploreBottomPanel` ou `ExploreAnonymousBottomPanel` (pas de prop à faire passer)
- Support de plugins remark/rehype au-delà du rendu de base

## Decisions

**Hook partagé `useExploreViewMode`**
Encapsule la lecture/écriture localStorage dans un hook React custom plutôt que de dupliquer la logique dans chaque panel.
Alternatives : state local sans persistance (rejeté — demande explicite localStorage) ; Zustand store (overkill pour une seule valeur booléenne).

**Clé localStorage `explore-view-mode`**
Namespacing minimal, suffisant vu qu'il n'y a qu'un seul toggle de ce type dans l'app.

**Défaut `raw`**
Décision produit (explore mode = inspection du contenu brut par défaut). `DetailPanel` défaut à `rendered` mais c'est un contexte différent (lecture de documents).

**Messages partiels rendus en mode `rendered`**
Risque accepté de layout instable pendant le streaming. Évite une logique conditionnelle supplémentaire et des flashs raw→rendered à la fin du stream.

**UI du toggle : icônes `Code` / `Eye`**
Cohérence avec `DetailPanel`. Groupées dans un petit container `bg-slate-100 rounded-md p-0.5` dans le header, entre le statut de connexion et le bouton de fermeture.

**Classes prose pour ReactMarkdown**
`prose prose-slate prose-sm max-w-none text-left` — même setup que `DetailPanel`. Le `max-w-none` est important dans des panels à largeur fixe.

## Risks / Trade-offs

- **Markdown partiel instable** → Risque accepté. Les headers et blocs de code incomplets peuvent provoquer des sauts de layout pendant le streaming. Atténuation possible dans une version future : forcer raw sur `msg.partial === true`.
- **localStorage silencieux** → En mode privé ou avec storage bloqué, le hook doit fallback sur le défaut `raw` sans crasher. Le hook doit gérer l'exception `try/catch`.
- **Synchronisation multi-onglets** → Non géré. Si l'utilisateur change le toggle dans un onglet, l'autre ne se met pas à jour. Acceptable pour ce cas d'usage.
