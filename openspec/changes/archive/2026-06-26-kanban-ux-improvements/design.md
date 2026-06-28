## Context

Le Kanban Board est implémenté en React (TypeScript) avec des styles inline. Le backend est en Go (chi router). Trois problèmes indépendants sont traités en un seul change car ils concernent tous la même page et partagent des modifications de fichiers.

État actuel :
- `KanbanPage` gère un seul état `exploreChange: string | null` pour l'ExplorePanel
- `ChangeCard` affiche des boutons d'action directement sur la carte
- Le layout utilise `alignItems: flex-start` (colonnes ne s'étirent pas) et le div flex-row manque de `width: 100%`
- Aucun endpoint de détail d'un change individuel

## Goals / Non-Goals

**Goals:**
- Rendre les cartes entièrement cliquables avec comportement différencié par colonne
- Créer un `DetailPanel` riche pour les colonnes To Do / In Progress / Done
- Épurer les cartes (suppression des boutons inline)
- Corriger le layout hauteur et largeur du Kanban

**Non-Goals:**
- Édition des tâches depuis le DetailPanel (lecture seule)
- Pagination ou virtualisation des colonnes
- Persistance de l'état du panneau ouvert après rechargement

## Decisions

### 1. État du panneau actif : union type plutôt que deux flags séparés

```typescript
type ActivePanel =
  | { type: 'explore'; name: string }
  | { type: 'detail'; name: string }
  | null
```

Alternatif considéré : deux états distincts `exploreChange` et `detailChange`. Rejeté car impossible d'ouvrir les deux simultanément et les deux états peuvent se désynchroniser.

Le type union garantit l'exclusivité et simplifie les conditions de rendu.

### 2. Propagation des événements : stopPropagation sur les boutons internes

La carte a un `onClick` global. Les boutons internes (dans le DetailPanel migré) n'ont plus besoin de `stopPropagation` car ils sont dans le panneau, pas sur la carte. Pendant la transition drag-start, `onDragStart` reste sur la carte — aucun conflit avec `onClick` car le navigateur distingue drag et click natifs.

### 3. Endpoint détail : GET /changes/{name} retourne tasks + artifacts

```json
{
  "name": "...",
  "kanban_status": "...",
  "tasks_done": 0,
  "tasks_total": 5,
  "created": "...",
  "tasks": [
    { "text": "Faire X", "done": false }
  ],
  "artifacts": {
    "proposal": "<contenu markdown>",
    "design": "<contenu markdown>"
  }
}
```

Alternatif : deux endpoints séparés `/tasks` et `/artifacts`. Rejeté car le DetailPanel a besoin des deux et un seul appel est plus simple côté frontend (un seul état de loading).

Les artifacts retournés sont limités à `proposal.md` et `design.md` — `tasks.md` est déjà parsé dans le champ `tasks`. Fichiers absents → champ `null` ou chaîne vide, pas d'erreur.

### 4. Layout : alignItems stretch + width 100% sur le conteneur flex

Dans `KanbanPage`, le div flex-row des colonnes reçoit :
- `alignItems: 'stretch'` (au lieu de `flex-start`) → colonnes pleine hauteur
- `width: '100%'` → remplit tout l'espace disponible

Dans `KanbanColumn`, `minHeight: 200` est remplacé par `alignSelf: 'stretch'` (implicite via `stretch` parent). La colonne garde `flex: 1` pour la distribution de largeur.

`KanbanPage` passe de `overflow: 'auto'` à `overflow: 'hidden'` sur le conteneur externe et ajoute `overflowY: 'auto'` sur le div flex-row pour un scroll vertical si le contenu dépasse.

## Risks / Trade-offs

- [Lecture des fichiers artifacts côté backend] Fichiers volumineux (design.md long) sont chargés en mémoire entièrement → Acceptable pour l'usage actuel (docs de planning, jamais > quelques Ko)
- [Clic vs drag] Un utilisateur qui commence un drag sur une carte pourrait déclencher le click → Le navigateur ne fire pas `onClick` quand `mouseup` survient après un drag significatif ; comportement natif suffisant sans logic supplémentaire

## Open Questions

_(aucune — toutes les décisions ont été prises en session d'exploration)_
