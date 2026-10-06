# Design

## Context

Voir `proposal.md` pour la motivation. État actuel observé :

- `runWorker` (`internal/pool/worker.go`) enchaîne : provisionnement, démarrage du subprocess, tour `/opsx:apply`, `validateAndHeal` (build/tests avec guérison sur le même subprocess), contrôle de complétion (`ParseTaskProgress` : `done < total` → `pause("…tâches restantes incomplètes (%d/%d)…")`), `CommitAll`, puis finalisation (fusion en `full-autonomy`, `MarkReview` en `hitl-review`).
- Le prompt système de l'agent d'un worker passe par le paramètre `extraSystemPrompt` de `session.StartSubprocess`, aujourd'hui vide pour les workers ; la directive de langue a son propre paramètre et les deux sont assemblés (`joinPrompts`).
- `openspec.ParseTaskProgress` et `parseTaskList` ne reconnaissent que les lignes `- [ ]` / `- [x]` ; le texte d'une tâche est la fin de la ligne telle quelle (un commentaire HTML y serait affiché).
- `ApproveReview` (`internal/pool/review_actions.go`) : `reviewLock` → `checkReviewable` → court-circuit si la branche est déjà fusionnée → `Provision` → intégration, validation et fusion. Les erreurs sont traduites en codes par `handlers/review_actions.go` (`reviewActionError{Code, Message}`).
- Le DetailPanel affiche le bouton « Approuver & Fusionner » dans l'onglet Actions ; `KanbanPage` ouvre `ApproveDialog` au drop To Review vers Done ; les erreurs de revue passent par `REVIEW_ERROR_CODES` et `reviewErrors.*` (`lib/api.ts`, locales `dialogs`).
- Ce change suppose livré `tasks-follow-the-branch` : lecture des tâches de la branche en revue (`WorktreeController.BranchTasks`), toggle committé sur la branche, compteurs de carte issus de la branche.

## Goals / Non-Goals

**Goals:**
- Qu'un change dont il ne reste que des tâches de validation humaine passe en To Review au lieu de bloquer en pause.
- Garder le filet de sécurité : une tâche oubliée par l'agent (non marquée) continue de bloquer.
- Que l'approbation exige que l'utilisateur ait déclaré avoir validé (tâches cochées).
- Que le contrat (marqueur, triage, directive) vive dans l'application et pas dans des skills externes.

**Non-Goals:**
- Un verrou technique contre l'abus du marqueur par l'agent.
- Lancer l'application depuis la revue.
- Rendre « Reprendre en finalisant » sensible au marqueur.
- Un badge « à valider » sur la carte du Kanban.

## Decisions

### D1. Le marqueur est un commentaire HTML sur la ligne de la tâche, lu par le paquet `openspec`

`<!-- human review required -->` est reconnu par une expression régulière insensible à la casse et aux espaces. `parseTaskList` retire le commentaire du texte et renseigne `Task.HumanReview` (`human_review,omitempty`). Une fonction `ParseTaskStats` (et sa variante sur contenu, utilisée avec `BranchTasks`) renvoie `Done`, `Total`, `PendingHuman` (non cochées marquées) et `PendingOther` (non cochées sans marqueur). `ParseTaskProgress` garde son contrat : les compteurs exposés comptent toutes les tâches. Le toggle ne touche que le préfixe `- [ ]` / `- [x]` : le commentaire est conservé, et le texte utilisé dans l'entrée d'activité est nettoyé.

Un commentaire HTML est invisible au rendu Markdown et inerte pour le CLI OpenSpec (qui ne regarde que les cases à cocher), et une ligne reste lisible à la main.

*Alternatives :* (a) une section dédiée `## Validation` : impose une structure que les skills ne produisent pas, et ne dit rien d'une tâche isolée ; (b) un préfixe `[manual]` visible dans le texte : pollue l'affichage et casse le format attendu par d'autres outils ; (c) un fichier annexe : une seconde source de vérité à synchroniser.

### D2. Ordre du worker : application, triage, validation, complétude, commit

```
 /opsx:apply
     |
     v
 hitl-review ET tâches non cochées sans marqueur ?
     | oui                                  | non
     v                                      |
 tour de triage (un seul) ------------------+
     |
     v
 validateAndHeal  ->  contrôle de complétion  ->  CommitAll  ->  MarkReview
```

Le triage précède la validation : si l'agent termine une tâche pendant le triage (du code change), la validation rejoue sur le résultat. Il réutilise le même subprocess et `runTurn`, comme le tour de guérison. Le prompt du tour liste les tâches concernées (texte sans marqueur) et demande, pour chacune, de la terminer et de la cocher, ou de lui ajouter `<!-- human review required -->` en fin de ligne sans la cocher ; il précise qu'aucun autre fichier ne doit être modifié pour ce seul marquage. Un échec du tour suit `agentTurnPauseReason` (pause, raison lisible).

Après le tour, le worker relit `tasks.md`, compare l'ensemble des tâches marquées avant et après, et consigne une entrée d'activité `pool.task_flagged` (catégorie `pool`, résumé `Task flagged for human review: <texte>`, meta `worker_id` et `change`) par tâche nouvellement marquée. Les tâches marquées avant le triage ne produisent pas d'entrée.

### D3. Le contrôle de complétion dépend du mode de délégation

Une fonction du paquet `pool` décide si des tâches « restantes » bloquent : en `full-autonomy`, `Done < Total` (comportement actuel, le marqueur est ignoré) ; en `hitl-review`, `PendingOther > 0`. Le message de pause reste « Validation réussie mais tâches restantes incomplètes (done/total) dans tasks.md ». Si des tâches non marquées subsistent après le triage, le worker se met en pause comme aujourd'hui : c'est le repli sûr si l'agent ignore la consigne.

Le même contrôle sert à la reprise après correction (un worker `RoleFixer` suit le flux normal) : les tâches marquées restées décochées ne bloquent pas le retour en revue.

*Alternative :* ne plus exiger de complétion en `hitl-review` (tolérance totale). Écartée : une tâche réellement oubliée passerait en revue sans signal ; le marqueur est ce qui permet de la distinguer d'une tâche humaine.

### D4. La directive passe par le prompt système, uniquement en `hitl-review`

Une constante du paquet `pool` contient la directive (en anglais, comme les prompts d'agent) : ne jamais cocher une tâche portant `<!-- human review required -->`, et marquer plutôt que cocher une tâche qui exige un humain. Elle est passée comme `extraSystemPrompt` au démarrage du subprocess pour les workers `hitl-review` ; elle reste vide en `full-autonomy`. Elle couvre aussi le cas des tâches humaines déjà écrites par `/opsx:ff` sans marqueur, que le tour d'application pourrait sinon cocher sans les avoir réalisées ; le triage ne s'en chargerait alors jamais puisqu'il ne reste rien de décoché.

*Alternative :* ajouter la consigne au texte du tour `/opsx:apply` : mélange une commande slash et du texte libre, et certaines commandes d'agent n'acceptent pas d'arguments supplémentaires. Le prompt système est le canal déjà prévu.

### D5. L'approbation vérifie les tâches de la branche, sous le verrou de revue, avant tout le reste

Dans `ApproveReview`, après `checkReviewable` et le court-circuit « branche déjà fusionnée » (la reprise d'un nettoyage ne doit pas être bloquée), et avant `Provision` et l'intégration : lecture du `tasks.md` de la branche (`BranchTasks`), calcul des tâches non cochées ; s'il y en a, retour de `*TasksPendingError{Remaining}`. Le handler le traduit en `409` `tasks_pending` avec `remaining`. Tout tâche non cochée compte, marquée ou non : si l'utilisateur décoche une tâche faite par l'agent, l'approbation est bloquée aussi (il a déclaré qu'elle n'est pas validée). Un `tasks.md` absent ou sans tâche dans la branche ne bloque pas.

Le verrou est celui qui sérialise déjà les actions de revue ; le toggle de `tasks-follow-the-branch` prend ce verrou en `TryLock`, donc une coche concurrente est soit avant l'approbation (prise en compte), soit refusée (`review_busy`).

### D6. Interface : le bouton reste visible et désactivé, le drop est refusé en amont

- `DetailPanel` : le nombre de tâches à valider est `data.tasks.filter(t => !t.done).length` (les tâches du détail viennent de la branche en revue). Si > 0, le bouton est désactivé avec `title` = « N tâche(s) à valider ». Les tâches marquées portent un badge ; l'onglet Tâches affiche un message de décompte en revue.
- `KanbanPage` : avant d'ouvrir `ApproveDialog` pour un drop To Review vers Done, `change.tasks_done < change.tasks_total` refuse le drop avec une notification. Les compteurs de carte viennent de la branche (`tasks-follow-the-branch`) ; en cas de compteurs périmés, le `409` `tasks_pending` du backend, ajouté à `REVIEW_ERROR_CODES` et à `reviewErrors`, reste le garde-fou.
- Libellés fr/en dans `dialogs` et `detailPanel` ; `ApproveDialog` n'a pas besoin de changer.

## Risks / Trade-offs

- [L'agent abuse du marqueur pour s'épargner du travail] → pas de verrou technique ; la validation (build/tests) reste un garde-fou, chaque tâche marquée par le triage est tracée dans l'activité, les tâches s'affichent avec un badge en revue et Approve exige que l'utilisateur les coche une à une.
- [L'agent ignore la consigne de triage] → les tâches non marquées subsistent et le worker se met en pause comme aujourd'hui : aucune régression, seulement un cas de plus géré correctement.
- [Un tour d'agent en plus] → uniquement en `hitl-review` et quand des tâches non cochées sans marqueur subsistent après l'application.
- [Les compteurs de carte et la liste des tâches dépendent de `tasks-follow-the-branch`] → dépendance déclarée ; sans lui, Approve se baserait sur le `tasks.md` du dépôt principal. L'implémentation de ce change doit venir après.
- [« Reprendre en finalisant » exige toujours que tout soit coché, y compris les tâches marquées] → voie de secours après une pause ; l'utilisateur coche les tâches dans le worktree (comportement archivé). Alignement possible plus tard sans changer les formats.
- [Le décompte d'Approve inclut les tâches décochées par l'utilisateur] → voulu (D5).
- [Une tâche humaine déjà cochée à tort par l'agent avant le triage] → non détectable sans verrou ; la directive de D4 réduit le risque.

## Migration Plan

Aucune migration de données : les `tasks.md` existants sans marqueur se comportent comme avant (toute tâche non cochée bloque). Retour arrière : revert du change ; les marqueurs déjà écrits dans des `tasks.md` sont des commentaires inertes.
