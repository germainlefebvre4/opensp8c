## Context

Le Kanban comporte deux panels latéraux : `DetailPanel` (colonnes To Do/In Progress/Done) et `ExplorePanel` (colonne To Explore). Les deux sont actuellement `position: fixed` — ils flottent par-dessus les colonnes, les masquant partiellement ou totalement. `KanbanPage` les monte dans le DOM mais ils n'occupent aucun espace dans le flux.

Le `DetailPanel` affiche les artifacts `proposal.md` et `design.md` dans un `<pre>` brut — illisible pour du contenu Markdown long. `react-markdown` et `@tailwindcss/typography` sont déjà installés (cf. `ui-modernize`).

## Goals / Non-Goals

**Goals:**
- Rendre les panels non-overlay : le Kanban et le panel partagent l'espace horizontal
- Même traitement pour DetailPanel et ExplorePanel
- Permettre de lire les artifacts Proposal et Design en Markdown rendu dans le DetailPanel (toggle shared Raw/Rendu)

**Non-Goals:**
- Redimensionnement du panel par drag (pas de splitter)
- Persistance du mode Raw/Rendu entre sessions
- Refactoring des panels au-delà du layout et du rendu Markdown

## Decisions

### 1. Layout KanbanPage : flex horizontal avec slot conditionnel

`KanbanPage` adopte un layout `flex flex-row` :
- Zone colonnes : `flex-1 overflow-x-auto` (prend tout l'espace si pas de panel, se rétrécit si panel ouvert)
- Slot panel : `w-[420px] shrink-0` conditionnel — présent uniquement si un panel est actif

```
Panel fermé          Panel ouvert
┌──────────────────┐  ┌─────────────┬──────────┐
│  colonnes (100%) │  │ colonnes    │  panel   │
│                  │  │ (flex: 1)   │  (420px) │
└──────────────────┘  └─────────────┴──────────┘
```

**Alternative rejetée** : `margin-right` dynamique sur le container colonnes — plus complexe, animation saccadée.

### 2. DetailPanel et ExplorePanel deviennent des flex children

Suppression de `position: fixed`, `top/bottom/right`, `z-index`, `boxShadow` côté positionnement (on garde la shadow gauche `border-l`). Les panels passent à `h-full flex flex-col` — ils remplissent leur slot.

### 3. Toggle Raw/Rendu : état partagé dans DetailPanel, visible uniquement sur onglets Proposal/Design

Un état `viewMode: 'raw' | 'rendered'` dans `DetailPanel`. Le toggle (icône `Code` / `FileText` de lucide-react) s'affiche dans la barre de tabs uniquement quand l'onglet actif est `proposal` ou `design`. Le mode s'applique aux deux onglets (basculer en Rendu sur Proposal → Design s'affiche aussi en Rendu).

**Alternative rejetée** : mode indépendant par onglet — surcharge cognitive inutile.

### 4. Rendu Markdown dans DetailPanel

Même pattern que `SpecsPage` : `<ReactMarkdown>` avec la classe `prose prose-slate prose-sm`. Pas de TOC (panel trop étroit). Pas de heading IDs (pas de navigation nécessaire).

## Risks / Trade-offs

- **Colonnes plus étroites quand panel ouvert** : avec un panel de 420px, sur un écran 1280px il reste ~860px pour 4 colonnes soit ~215px chacune — proche du minimum viable. Le `overflow-x-auto` sur la zone colonnes protège contre le débordement. → Acceptable, et l'utilisateur peut passer en fullpage (feature existante de `ui-modernize`).
- **ExplorePanel** : son contenu (chat) est vertical — pas affecté par la largeur réduite. La zone de saisie reste utilisable.
