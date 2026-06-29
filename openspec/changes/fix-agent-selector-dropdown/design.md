## Context

Le composant `AgentSelector` affiche un dropdown pour choisir l'agent CLI actif. Il est rendu dans `WorkspaceSidebar`, dont l'élément `<aside>` et son div interne portent tous deux `overflow-hidden` (nécessaire pour l'animation `transition-[width]`).

Problèmes actuels :
1. Le dropdown (`position: absolute`) déborde vers le bas et est clippé par `overflow-hidden` de la sidebar → seule une fine barre de 8px est visible
2. Le containing block a `pb-2` → `top-full` positionne le dropdown 12px sous le bouton au lieu d'y être collé
3. `agents.Detect()` exécute `exec.Command(...).Output()` sans timeout → si `gh copilot --version` bloque, `/api/agents` ne répond jamais → dropdown toujours vide

## Goals / Non-Goals

**Goals:**
- Dropdown visible et correctement aligné sous le bouton
- Endpoint `/api/agents` répond toujours en moins de 5s

**Non-Goals:**
- Refactoring global de la sidebar
- Ajout de dépendances (pas de floating-ui / popper.js)
- Gestion du scroll de la page (aucun scroll global dans l'app)

## Decisions

### position: fixed avec getBoundingClientRect()

Le dropdown passe de `position: absolute` à `position: fixed` avec coordonnées calculées depuis `buttonRef.current.getBoundingClientRect()` au clic.

Alternatives considérées :
- **React Portal** : résout le clipping mais complexifie la gestion outside-click (le dropdown sort du DOM du ref)
- **Restructurer le sidebar** : les deux conteneurs ont `overflow-hidden`, déplacer `AgentSelector` ne suffit pas
- **Supprimer overflow-hidden** : casse l'animation de largeur

`position: fixed` est positionné relativement au viewport, donc **immune à tout `overflow: hidden`** sur les ancêtres (sauf `position: fixed` eux-mêmes, ce qui n'est pas le cas ici). C'est la solution la moins invasive.

### Coordonnées calculées au clic uniquement

La position est calculée à l'ouverture. Un `useEffect` écoute `window.resize` pour fermer le dropdown si la fenêtre est redimensionnée (position deviendrait stale). Le scroll n'est pas géré (pas de scroll global dans l'app).

### Timeout 3s sur exec.Command dans Detect()

Utiliser `context.WithTimeout(context.Background(), 3*time.Second)` + `exec.CommandContext`. Si une CLI ne répond pas en 3s, elle est marquée `installed: false`. Valeur choisie : assez courte pour ne pas bloquer l'UI, assez longue pour les CLIs légitimement lentes au démarrage.

## Risks / Trade-offs

- **Position stale après resize** → mitigation : fermer le dropdown sur `window.resize`
- **Timeout 3s trop court pour certaines CLIs** → acceptable : une CLI qui prend >3s à afficher sa version n'est pas utilisable en pratique
- **getBoundingClientRect() retourne {0,0} si bouton non visible** → impossible : le bouton doit être visible pour être cliqué
