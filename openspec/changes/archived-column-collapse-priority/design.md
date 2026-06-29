## Context

Le slot partagé Done/Archived dans `KanbanPage.tsx` utilise un layout `flex-col`. Done porte `flex-1 min-h-0` (s'étire et rétrécit), Archived porte `shrink-0` (ne rétrécit jamais). Résultat : tout rétrécissement vertical est absorbé par Done — à l'inverse de la priorité souhaitée.

`KanbanColumn` expose déjà un prop `maxVisible` et un mécanisme "Afficher plus". Il n'existe pas de bouton collapse.

## Goals / Non-Goals

**Goals:**
- Archived cède l'espace vertical en premier ; Done occupe tout l'espace résiduel
- La colonne Archived peut être entièrement réduite (collapsed) via un bouton dans son header
- `maxVisible` de la colonne Archived passe à 3 par défaut

**Non-Goals:**
- Redimensionnement dynamique de `maxVisible` selon la hauteur disponible (ResizeObserver)
- Persistance de l'état collapsed entre sessions

## Decisions

### D1 — Priorité flex via `max-h` sur Archived

**Choix** : Archived reçoit `max-h-[40%] overflow-y-auto` à la place de `shrink-0`. Done conserve `flex-1 min-h-0`.

**Pourquoi** : `shrink-0` empêche toute compression. Plafonner Archived à 40 % de la hauteur du parent garantit que Done dispose toujours d'au moins 60 % de l'espace. La valeur 40 % est un compromis lisible (≈ 4–5 cartes sur un écran standard) sans nécessiter de ResizeObserver.

**Alternative rejetée** : `flex-shrink: 2` sur Archived pour une compression proportionnelle — plus élégant en théorie, mais imprévisible selon le nombre de cartes dans Done.

### D2 — État collapse dans `KanbanColumn`

**Choix** : Ajout d'un `useState<boolean>(false)` local `collapsed` dans `KanbanColumn`, visible uniquement si une nouvelle prop `collapsible?: boolean` est passée à `true`. Un chevron dans le header toggle l'état. Quand `collapsed === true`, la liste de cartes et le bouton "Afficher plus" sont masqués (`display: none` ou retrait conditionnel du JSX).

**Pourquoi** : L'état collapse est éphémère (reset à chaque montage), local à la colonne, sans impact sur le reste du board. Un prop opt-in évite de toucher au comportement des autres colonnes.

**Alternative rejetée** : Gérer l'état dans `KanbanPage` — surcharge inutile, pas de bénéfice inter-composant.

### D3 — `maxVisible` à 3

**Choix** : `maxVisible={3}` passé depuis `KanbanPage` pour Archived (au lieu de 5).

**Pourquoi** : 3 cartes représentent l'empreinte minimale utile (contexte récent) sans occuper trop de hauteur par défaut. L'utilisateur peut toujours afficher plus.

## Risks / Trade-offs

- `max-h-[40%]` est relatif au parent direct (`flex-col` du slot) — si ce parent change de structure, la valeur devra être recalculée. → Mitigation : commentaire inline sur la valeur.
- Sur des écrans très petits (< 600px de hauteur), 40 % peut toujours créer une compression visible de Done. → Acceptable pour l'usage bureau ciblé.
- Le scroll interne d'Archived (`overflow-y-auto`) crée une double scrollbar si le contenu est déjà limité par `maxVisible`. → Pas de problème en pratique : le scroll n'apparaît que si le contenu dépasse le plafond, ce qui suppose un grand nombre de cartes après "Afficher plus".
