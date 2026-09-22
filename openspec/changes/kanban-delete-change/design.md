# Design

## Context

Voir `proposal.md` pour le "pourquoi". Contraintes techniques observées dans le code existant :

- Aucun endpoint backend ne supprime un change aujourd'hui. `archive.go` délègue à la CLI `openspec archive <name> --yes` (`exec.Command`), mais il n'existe pas d'équivalent CLI pour une suppression définitive : ce sera une opération filesystem directe (`os.RemoveAll`).
- Le seul flux de suppression existant est celui des ghost records (`explore.go:DeleteGhost`) : `mgr.StopAnonymous`, `prefs.DeleteExploration(ghostID)`, suppression du fichier de brouillon, `convStore.DeleteExplorationLogs`, puis broadcast `exploration_deleted`. On réutilise cette même séquence pour la cascade.
- L'association ghost <-> change solide n'est pas une clé étrangère stockée : le frontend la déduit en comparant les noms (`KanbanColumn.tsx`: `allChanges.find(c => c.is_ghost && c.name === ch.name)`). Le backend doit faire la même chose : chercher, parmi `prefs.ListExplorations(workspaceID)`, un enregistrement dont `.Name == changeName`.
- `pool.Manager.Status()` retourne `(AgentPoolConfig, bool, []Worker)`, où chaque `Worker` a un champ `ActiveChange`. C'est la source de vérité pour savoir si un worker est actif sur le change qu'on veut supprimer — pas besoin de nouveau mécanisme de suivi.
- `DetailPanel.tsx` gère déjà un `type Tab` fermé et des footers conditionnés par `data.kanban_status` (`=== 'done'`, `=== 'to-review'`). Le nouvel onglet Actions remplace ces deux footers par un rendu conditionnel à l'intérieur du contenu de l'onglet.
- `ResetTasksDialog.tsx` est le seul pattern de confirmation modale existant côté frontend (overlay `fixed inset-0`, carte blanche centrée, boutons Annuler/Confirmer) — réutilisé tel quel pour `DeleteChangeDialog`.

## Goals / Non-Goals

**Goals:**
- Suppression définitive et sûre d'un change depuis le Kanban, avec garde-fou contre les workers actifs et cascade sur le ghost associé.
- Un seul point d'entrée UI (onglet Actions) pour toutes les actions de cycle de vie du DetailPanel.

**Non-Goals:**
- Implémenter le flux "Approuver & Fusionner" / "Demander correction" (boutons déplacés tels quels, toujours sans handler).
- Implémenter les boutons génériques de transition de statut ("→ In Progress") déjà décrits par le requirement existant "Changer le statut depuis le DetailPanel" — ce requirement n'est pas touché par ce change.
- Mécanisme de corbeille ou de restauration : la suppression est irréversible (décision actée pendant l'exploration).
- Arrêter/annuler un worker actif pour permettre la suppression : on se contente de désactiver le bouton (décision actée pendant l'exploration).

## Decisions

**Suppression directe du dossier, pas de délégation CLI.** Contrairement à l'archivage, il n'existe pas de sous-commande `openspec` pour supprimer un change. Le handler appelle directement `os.RemoveAll(filepath.Join(workspacePath, "openspec", "changes", name))`, après validation d'existence (`os.Stat`) et de l'absence de worker actif.

**Détection du worker actif via `pool.Manager.Status()`.** Le `KanbanHandler` reçoit une référence vers `*pool.Manager` (comme `PoolHandler` l'a déjà). Avant suppression : `_, _, workers := mgr.Status()`, puis recherche d'un `Worker` avec `ActiveChange == name`. Si trouvé → 409. Alternative écartée : ajouter un champ `locked` dans `.openspec.yaml` — plus de surface de synchronisation à maintenir pour un état déjà disponible en mémoire dans le Manager.

**Cascade du ghost par correspondance de nom.** Même heuristique que le frontend (`e.Name == changeName` parmi `prefs.ListExplorations(workspaceID)`), pour rester cohérent avec l'affichage de la bannière "Brouillon d'exploration actif" que l'utilisateur a vue avant de cliquer Supprimer. Si un ghost correspondant existe, on réexécute la séquence de `DeleteGhost` (stop session, `DeleteExploration`, suppression du brouillon, `DeleteExplorationLogs`) avant de retourner 204.

**Un seul composant `DeleteChangeDialog`, calqué sur `ResetTasksDialog`.** Mêmes classes Tailwind, même structure (titre, corps, Annuler/Confirmer), pour rester visuellement cohérent sans introduire de nouvelle primitive de modal.

**L'onglet Actions absorbe les deux footers conditionnels existants plutôt que de les dupliquer.** Le rendu conditionnel (`data.kanban_status === 'done'` / `=== 'to-review'`) se déplace du niveau JSX "footer" (`DetailPanel.tsx:389-403` et `:374-387`) vers l'intérieur du bloc `activeTab === 'actions'`. Le bouton Supprimer, lui, est présent pour tout statut différent de `archived` (et n'existe pas comme footer aujourd'hui, il est purement nouveau).

**Le bouton "Sync & Archive" de `ChangeCard.tsx` (hover, statut Done) n'est pas touché.** Décision actée pendant l'exploration : il coexiste avec le bouton Archiver de l'onglet Actions, comme raccourci rapide directement sur la carte.

## Risks / Trade-offs

- **[Suppression irréversible]** → Confirmation modale obligatoire avant tout appel DELETE ; aucune atténuation supplémentaire (corbeille) n'est dans le périmètre, décision actée pendant l'exploration.
- **[Race entre la vérification "worker actif" et le lancement effectif d'un worker]** → Fenêtre de course théorique entre le check `mgr.Status()` et le `RemoveAll` si le dispatcher démarre un worker sur ce change entre-temps. Le dispatcher pioche dans la colonne Todo (`agent-pool-orchestrator`) ; le risque existe surtout pour les changes en To Do avec pool actif. Non traité ici (le pool est encore un stub sans intégration UI complète, cf. `agent-pool-orchestrator`/`agent-pool-ui` à 0 requirement) — à revisiter quand le pool sera réellement branché sur le dispatch automatique.
- **[Deux boutons Archiver après ce change — carte et onglet Actions]** → Redondance assumée (décision actée pendant l'exploration) ; les deux appellent le même hook `useArchive`, donc pas de divergence de comportement possible.

## Open Questions

Aucune — les décisions de périmètre (statuts autorisés, cascade ghost, garde-fou worker, contenu de l'onglet Actions, sort des boutons To Review) ont toutes été tranchées pendant l'exploration.
