## Context

Le Kanban actuel expose 4 colonnes dérivées dynamiquement du statut des tasks. Les changements archivés (`openspec/changes/archive/`) sont ignorés par le backend (`ListChanges` skippe explicitement le répertoire `archive`). L'action d'archivage existe déjà côté backend (`POST /changes/{name}/archive`) et appelle `openspec archive <name> --yes`.

## Goals / Non-Goals

**Goals:**
- Afficher les changements archivés dans une 5e colonne Archived
- Exposer l'action archive comme quick-action au survol des cartes Done
- Distinguer visuellement actif vs archivé (séparateur + style muted)
- Pagination de la colonne Archived (5 par défaut, +5 par clic)

**Non-Goals:**
- Désarchivage depuis l'UI
- Suppression de changements archivés
- Archivage groupé (pour plus tard)
- Sync IA (opsx:sync) comme étape préalable — `openspec archive --yes` suffit

## Decisions

### D1 — Endpoint dédié pour les changements archivés

Nouveau `GET /workspaces/{id}/archived-changes` plutôt qu'un query param sur l'endpoint existant.

**Pourquoi** : l'endpoint actuel retourne `[]Change` avec polling à 5s. Les archivés changent rarement — les séparer permet un polling moins fréquent côté frontend (ou aucun polling, refetch uniquement après archivage).

**Alternative écartée** : `?include_archived=true` sur `/changes`. Aurait mélangé données actives et archivées dans un seul tableau, complexifiant le filtrage côté frontend sans gain.

### D2 — `kanban_status: "archived"` retourné par le backend

`ListArchivedChanges()` lit `openspec/changes/archive/` et retourne des `Change` avec `kanban_status = "archived"`. Le `deriveStatus` existant n'est pas modifié — les archivés ont un statut fixe indépendant de leurs tasks.

**Pourquoi** : cohérence avec la structure de données existante, zéro changement dans les composants qui consomment `Change`.

### D3 — Quick-action "Sync & Archive" sur la carte Done (hover)

Bouton visible uniquement au survol des cartes en colonne `done`, réutilisant l'endpoint `/archive` existant.

**Pourquoi** : l'exploration a confirmé que `openspec archive --yes` est déjà non-interactif et gère la sync des specs automatiquement. Pas besoin d'un flow à deux étapes.

**Nom "Sync & Archive"** : communique explicitement que les specs sont synchronisées en même temps que l'archivage, ce que le label "Archiver" seul ne transmettait pas.

### D4 — Pagination locale (pas de pagination backend)

La colonne Archived charge tous les changements archivés en une requête, et la pagination (5/+5) est gérée côté frontend avec un state `visibleCount`.

**Pourquoi** : le nombre de changements archivés reste raisonnable à moyen terme. La complexité d'une pagination backend (cursors, offset) n'est pas justifiée.

**Limite** : si l'archive grossit à plusieurs centaines de changements, une pagination backend sera nécessaire. Le non-goal est acté et documenté.

### D5 — Pas de polling sur les changements archivés

`useArchivedChanges` n'a pas de `refetchInterval`. Refetch uniquement déclenché après un archivage réussi (invalidation de la query `archived-changes`).

**Pourquoi** : les changements archivés ne changent que lors d'une action explicite depuis l'UI. Un polling à 5s serait inutile.

## Risks / Trade-offs

- **Archive grandissante** → pagination backend à prévoir si le volume dépasse quelques dizaines. Mitigé par le "Afficher plus" qui retarde la perception du problème.
- **Pas de désarchivage UI** → si un changement est archivé par erreur, il faut passer par le CLI. Acceptable pour l'instant, le désarchivage est rare.
- **Conflit de spec existant** : `kanban-board` spec dit "pas de boutons inline sur les cartes" mais `change-archive` spec dit "bouton dans la carte". Ce delta spec résout la contradiction en explicitant que l'exception s'applique aux cartes Done (hover).

## Migration Plan

1. Ajouter `ListArchivedChanges()` et la route dans le backend (pas de migration de données)
2. Déployer le backend (l'endpoint `/changes` existant est inchangé)
3. Déployer le frontend avec la nouvelle colonne et le quick-action

Pas de rollback plan nécessaire — changement purement additif, aucune donnée existante modifiée.

## Open Questions

_(aucune — toutes les décisions ont été prises en explore session)_
