## Context

La page Specs (`SpecsPage.tsx`) est actuellement read-only : elle charge le contenu d'un `spec.md` via GET et le rend avec `ReactMarkdown`. Il n'existe aucun endpoint d'écriture ni surveillance du répertoire `openspec/specs/` par le watcher.

Le watcher surveille déjà `openspec/changes/**` avec un pattern `tryAddChangesDir` — il découvre les sous-dossiers au démarrage et les nouveaux à la création. Ce pattern est réutilisable pour `specs/`.

Le frontend SSE (`useWorkspaceEvents.ts`) traite déjà `change_created/updated/deleted` en invalidant React Query. Même mécanique pour `spec_updated`.

## Goals / Non-Goals

**Goals:**
- Mode édition inline sur SpecsPage avec split textarea / diff panel
- Save explicite (bouton + Ctrl+S) qui écrit sur disque via PUT
- Watcher étendu à `openspec/specs/` → SSE `spec_updated` → invalidation de cache
- Diff live (localContent vs savedContent) pour visualiser les modifications en cours
- Détection d'une modification externe pendant l'édition (warning, pas merge auto)

**Non-Goals:**
- Éditeur WYSIWYG ou syntax highlighting Markdown
- Auto-save ou draft persistant
- Résolution automatique de conflits
- Création / suppression de specs depuis l'UI
- Renommage de specs

## Decisions

### D1 — Fichier comme source de vérité (pas le state React)
Le save écrit sur disque. Le watcher détecte la modification, émet `spec_updated` via SSE, le frontend invalide la query et relit depuis le disque. L'UI ne met jamais à jour son cache React Query directement via la réponse PUT — c'est toujours le round-trip fichier → watcher → SSE → fetch qui confirme.

**Pourquoi** : cohérence avec le reste du système (changes utilisent le même pattern). Toute modification externe (CLI, éditeur) est automatiquement propagée.

**Alternatif écarté** : mise à jour optimiste du cache React Query directement depuis la réponse PUT. Plus rapide, mais crée une divergence possible si le watcher est décalé.

### D2 — Save explicite uniquement
Pas d'auto-save. Le save se déclenche sur bouton "Enregistrer" ou Ctrl+S.

**Pourquoi** : évite la boucle continue `frappe → save → watcher → SSE → re-fetch` qui perturberait l'édition. Un save explicite = un seul cycle.

### D3 — Diff view au lieu d'une preview markdown
Le panneau droit affiche `diffLines(savedContent, localContent)` depuis le package `diff`, pas un rendu ReactMarkdown.

**Pourquoi** : pour le cas d'usage principal (corriger une spec en cours de session), voir ce qui change est plus utile que voir le rendu. Le diff est la représentation de l'intention.

**Lib** : package npm `diff` + `@types/diff` (~7KB, zéro dépendances). `diffLines()` retourne des chunks `{added, removed, value}` — suffisant pour un affichage coloré.

### D4 — Deux états locaux disjoints
```
localContent  → nourrit la textarea et le calcul du diff
savedContent  → snapshote le dernier contenu confirmé par le serveur
```
Le re-fetch déclenché par `spec_updated` met à jour `savedContent`. Si `localContent !== savedContent` au moment du re-fetch (modification externe détectée), un warning est affiché — l'utilisateur décide.

### D5 — Pattern watcher symétrique à tryAddChangesDir
Nouvelle fonction `tryAddSpecsDir` dans `watcher.go` :
- Watch `openspec/specs/` au démarrage
- Watch chaque sous-dossier existant
- Dans `handleEvent` : un `CREATE` sur `specs/` ajoute dynamiquement le nouveau dossier ; un `WRITE` sur `specs/<name>/spec.md` déclenche un debounce 150ms → `spec_updated`

**Pourquoi** : réutiliser exactement le même pattern que `changes/` maintient la cohérence du code et évite une surface de bug différente.

### D6 — TOC masquée en mode édition
En mode édition, le panneau TOC est remplacé par le diff panel. Layout devient :
`[spec list | textarea | diff panel]` au lieu de `[spec list | content | TOC]`.

## Risks / Trade-offs

- **Boucle write → watcher → SSE → re-fetch en mode édition** → Mitigation : `savedContent` est mis à jour mais `localContent` (et donc la textarea) ne l'est pas. Le cycle est silencieux sauf si les deux divergent (modification externe).

- **Modification externe détectée pendant l'édition** → Mitigation : afficher un banner "Ce fichier a été modifié en dehors de l'éditeur — [Ignorer] [Écraser avec la version disque]". Pas de merge automatique.

- **Watcher ne surveille pas specs/ au démarrage si le dossier n'existe pas encore** → Mitigation : même stratégie que `changes/` — écouter la création de `specs/` dans `openspec/` via l'observation du répertoire parent.

- **Package `diff` ajoute une dépendance** → Trade-off assumé : l'alternative (implémenter un LCS manuellement) introduit plus de risque de bug qu'une lib testée. Taille négligeable (7KB).
