# Proposal

## Why

Aujourd'hui, dès qu'un Fast-Forward termine de générer `proposal.md`/`design.md`/`tasks.md` pour un change, celui-ci apparaît automatiquement dans la colonne **To Do** et devient *immédiatement* éligible au ramassage par l'Agent Pool (`GetRunnableChanges` ne filtre que sur le statut `todo`, sans notion de priorité, et pioche dans l'ordre alphabétique du répertoire). Il n'existe aucun moyen de garder une spec générée "en réserve" le temps de la prioriser, ni de choisir laquelle sera lancée en premier quand plusieurs changes attendent. Ajouter une colonne **Ready** entre **To Explore** et **To Do** comble ce trou : elle retient les changes dont les artefacts sont prêts mais dont le lancement du dev est différé, et permet de les ordonner par priorité.

## What Changes

- Ajout d'une colonne **Ready** entre **To Explore** et **To Do**.
- Le drag `To Explore → Ready` (et la promotion d'un ghost card) déclenche le Fast-Forward comme le fait aujourd'hui le drag vers `To Do` — mais la carte résultante atterrit en `Ready`, pas en `To Do`.
- Nouvelle action manuelle, réversible, `Ready ↔ To Do` (drag ou bouton) : ne touche pas `tasks.md`, ne fait que (dé)marquer le change comme "lancé". Seul un change "lancé" (`To Do` et au-delà) est éligible au ramassage par l'Agent Pool.
- Les cartes en colonne `Ready` sont réordonnables par drag-and-drop ; cet ordre est persisté et **BREAKING** pour le dispatcher : `agent-pool-orchestrator` ne pioche plus les changes `To Do` dans l'ordre alphabétique du répertoire mais dans l'ordre de priorité hérité de `Ready`.
- Le statut Kanban `todo` n'est plus dérivable uniquement de `tasks.md` (comme aujourd'hui via `deriveStatus`) : il dépend désormais aussi d'un marqueur "lancé" persistant par change. Les changes existants sans ce marqueur sont traités comme déjà "lancés" (compatibilité ascendante — pas de régression sur les changes en cours).
- Reset vers `To Explore` (drag `Ready → To Explore` ou `To Do → To Explore`) vide `tasks.md` **et** efface le marqueur de lancement et le rang de priorité, pour repartir d'un état propre.

## Capabilities

### New Capabilities

- `kanban-ready-column`: état persistant "lancé"/"rang de priorité" par change, colonne Ready, actions de promotion/rétrogradation `Ready ↔ To Do`, réordonnancement par drag-and-drop, règle de compatibilité ascendante pour les changes existants.

### Modified Capabilities

- `kanban-board`: le Kanban affiche six colonnes au lieu de cinq (`To Explore`, `Ready`, `To Do`, `In Progress`, `To Review`, `Done`/`Archived`) ; la dérivation du statut affiché pour un change avec `tasks.md` non démarré dépend désormais aussi du marqueur "lancé".
- `kanban-drag-drop`: la table des transitions de drag autorisées change (`To Explore → Ready` remplace `To Explore → To Do` pour le déclenchement direct du FF ; ajout de `Ready ↔ To Do` ; le reset `→ To Explore` est désormais possible depuis `Ready` en plus de `To Do`/`In Progress`).
- `exploration-promote-to-change`: la promotion d'un ghost card dépose désormais le change créé dans `Ready`, pas dans `To Do` ; les textes de dialogue référençant "To Do" sont mis à jour.
- `agent-pool-orchestrator`: le dispatcher DAG doit trier les changes `To Do` runnables selon leur rang de priorité persisté avant de les distribuer aux workers libres, au lieu de l'ordre de lecture du répertoire.
- `tasks-reset`: le reset (`PATCH .../tasks/reset`) efface aussi le marqueur "lancé" et le rang de priorité du change, en plus de vider `tasks.md`.
- `workspace-kanban-counts`: `task_counts` inclut désormais une clé `ready`.
- `explore-ghost-card`: la cible de drag autorisée pour un ghost card nommé devient la colonne `Ready` (au lieu de `To Do`).
- `explore-session`: le texte de comportement référençant le drag `to-explore → todo` est mis à jour vers `to-explore → ready`.
- `sidebar-done-badge`: la liste des pastilles de comptage inclut désormais `ready`.
- `stale-change-detection`: les changes en statut `ready` ne peuvent jamais être marqués `is_stale`, au même titre que `to-explore` et `todo`.

## Impact

- Backend : `internal/openspec/change.go` (`deriveStatus`, lecture/écriture `.openspec.yaml`), `internal/api/handlers/kanban.go` (nouvelles routes promote/demote/reorder, adaptation du reset), `internal/pool/scheduler.go` (tri par rang de priorité), `internal/api/handlers/workspace.go` (task_counts).
- Frontend : `KanbanPage.tsx` (colonnes, `VALID_DROPS`), `KanbanColumn.tsx`/`ChangeCard.tsx` (rendu de la colonne Ready, drag-to-reorder), `lib/api.ts` (nouveaux appels), locales `fr`/`en` `kanban.json`.
- Pas de migration de données destructive : les changes existants sans marqueur "lancé" restent visibles en `To Do` (comportement inchangé), seuls les nouveaux changes passeront par `Ready`.
