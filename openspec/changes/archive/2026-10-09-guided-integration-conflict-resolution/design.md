# Design

## Context

Voir `proposal.md` pour la motivation. État actuel observé :

- `IntegrateTarget` (`pool/worktree.go`) lance `git merge --no-edit <cible>` dans le worktree du change ; en cas d'échec il fait `merge --abort` et renvoie une erreur qui n'embarque que le `stderr`. Git écrit les fichiers en conflit sur **stdout** : le message est vide de toute information utile. `integrateAndMerge` (`pool/merge.go`) l'enveloppe dans `IntegrationConflictError{Target, Err}`, que le worker `full-autonomy` transforme en raison de pause et que l'approbation transforme en `409 integration_conflict` (`handlers/review_actions.go`, `approveFailure`).
- `RequestCorrection` → `applyCorrection` (`pool/review_actions.go`) ajoute **une** tâche `- [ ] Correction : …` dans `## Corrections` (première ligne du retour sur la ligne de tâche, le reste cité en `  > `), committe `tasks.md`, puis lève le marqueur ; `applyCorrection` est partagé avec la correction de la vérification (`verify_actions.go`). Aucun code ne décoche de tâche en dehors du reset vers To Explore.
- Le worker qui reprend une correction ignore les tâches marquées décochées dans son contrôle de complétion (`hitl-review`) et ne coche jamais une tâche marquée ; la vérification UI ne soumet à l'agent que les tâches marquées non cochées.
- Côté UI, `ApproveDialog` reste ouvert avec l'erreur (clé `reviewErrors.<code>` des locales) ; `DetailPanel` conserve aussi l'erreur dans l'onglet Actions ; `KanbanPage` ouvre le même `ApproveDialog` pour un drag vers Done et `CorrectionDialog` pour un drag vers In Progress. `CorrectionDialog` n'a ni texte initial ni option.

## Goals / Non-Goals

**Goals:**
- Que l'erreur `integration_conflict` dise quels fichiers sont en conflit, sans modifier l'état laissé par l'intégration annulée.
- Relier cette erreur au circuit de correction existant par un dialogue prérempli, relu et confirmé par l'utilisateur.
- Permettre de rouvrir les tâches de validation humaine cochées dans le même commit que la correction.

**Non-Goals:**
- Résoudre le conflit côté plateforme (pas de stratégie `ours`/`theirs`, pas de résolution automatique) : c'est le worker qui le fait, sur la branche du change.
- Traiter l'intégration de la branche cible **sans conflit** (coche humaine périmée après un merge propre de `main`) : limite connue, documentée.
- Changer la correction issue de la vérification (`verification/request-correction`) : elle n'expose pas l'option.
- Rouvrir automatiquement les tâches pour toute correction : le comportement par défaut d'une correction ordinaire reste celui d'aujourd'hui.

## Decisions

**1. Fichiers en conflit relevés avant `merge --abort`, par `git diff --name-only --diff-filter=U`.**
Dans `IntegrateTarget`, l'index du worktree contient les entrées non fusionnées au moment de l'échec ; on les lit juste avant d'annuler, puis on annule comme aujourd'hui. Les chemins sont relatifs à la racine du dépôt, triés. Si la lecture échoue, la liste est vide et l'erreur reste un `integration_conflict` (la résolution guidée reste proposée, sans liste).
*Alternative écartée* : `git merge-tree --write-tree --name-only` avant le merge. Il ne touche pas au worktree, mais ajoute un second calcul de merge, exige git ≥ 2.38 et peut diverger du vrai merge. Lire l'état réel du merge qu'on s'apprête à annuler est plus simple et exact.
`IntegrationConflictError` gagne un champ `Files []string` ; `Error()` ajoute la liste, ce qui profite aussi à la raison de pause du worker `full-autonomy`.

**2. Réponse enrichie, rétrocompatible.**
`reviewActionError` gagne `Target string` (`target`) et `Files []string` (`files`), omis quand vides. Le front lit `files` et `target` dans `ApiError` (le champ `target` existe déjà) ; les clients qui les ignorent ne sont pas affectés.

**3. La réouverture est une fonction pure sur le contenu de `tasks.md`, appliquée dans `applyCorrection`.**
Une fonction `ReopenHumanTasks(content) (updated string, reopened int)` du package `openspec` repasse `[x]`/`[X]` en `[ ]` sur les lignes de tâche portant le marqueur (même regex que `ParseTaskListContent`), sans toucher au reste de la ligne. `applyCorrection` reçoit un paramètre d'options ; avec `ReopenHumanTasks`, il applique d'abord la réouverture puis `AppendCorrection`, **avant** l'écriture : un seul `WriteFile`, un seul `CommitFile`, et la restauration de `original` existante couvre l'échec. `RequestCorrection` expose l'option ; la correction de vérification passe l'option à faux.
*Alternative écartée* : un second endpoint ou un second commit « Reopen task N » par tâche. Plus de commits, un état intermédiaire observable (tâches rouvertes sans correction ni marqueur levé) et deux chemins à garder cohérents.

**4. La réouverture ne dépend pas du nombre de tâches côté API.**
L'API accepte `reopen_human_tasks` quel que soit l'état et renvoie `204` même si rien n'est rouvert (scénario « sans tâche humaine cochée »). C'est le dialogue qui masque la case quand il n'y a rien à rouvrir.

**5. Le dialogue de correction devient configurable, pas dupliqué.**
`CorrectionDialog` reçoit `initialFeedback?`, `reopenDefault?` et le compte des tâches humaines cochées ; `onSubmit(feedback, reopenHumanTasks)`. Le compte est une prop (`humanTasksChecked`) calculée par le parent à partir de `useChangeDetail` (requête déjà en cache dans le `DetailPanel`, chargée à la demande depuis `KanbanPage`), pas d'un nouveau champ de l'API : le dialogue n'exige ainsi aucun `QueryClientProvider` et reste utilisable par `VerificationBanner`. `reopenDefault` vaut vrai uniquement pour l'ouverture depuis un conflit ; `useRequestCorrection` transmet l'option à `requestCorrection`.

**6. Résolution guidée : un composant d'indication partagé, ouvert depuis `ApproveDialog` et depuis l'erreur de l'onglet Actions.**
`ApproveDialog` mémorise `code`, `target` et `files` de l'erreur (en plus du message) et, pour `integration_conflict` uniquement, affiche la liste et un bouton fourni par une prop `onResolveConflict(target, files)`. Le parent ferme `ApproveDialog` et ouvre `CorrectionDialog` prérempli : `DetailPanel` pour l'onglet Actions, `KanbanPage` pour le drag vers Done (où `correctionDialog` existe déjà). L'erreur conservée par `DetailPanel` dans l'onglet Actions affiche la même liste et le même bouton, pour le cas où l'utilisateur a fermé le dialogue.

**7. Texte prérempli dans la langue de l'interface, première ligne autonome.**
`AppendCorrection` ne met que la première ligne sur la ligne de tâche et cite le reste. Le gabarit commence donc par une phrase complète (« Intégrer `main` dans la branche du change et résoudre les conflits, puis relancer la validation »), suivie de la liste des fichiers et de la consigne de conserver les évolutions des deux côtés. Il vit dans les locales `dialogs` (`reviewConflict.feedback…`), pas dans le backend : le backend reste agnostique de la formulation et l'utilisateur voit exactement ce qui sera écrit dans `tasks.md`.

**8. Aucun changement du pool, du worker ni du circuit de reprise.**
La résolution est un travail ordinaire du worker sur sa branche : `git merge <cible>` dans son worktree, résolution, validation, commit. Une fois la cible intégrée dans la branche, `TargetAhead` est faux à l'approbation suivante et l'intégration n'a plus lieu.

## Risks / Trade-offs

- [Le worker ne sait pas, ou ne peut pas, exécuter `git merge` dans son worktree] → Le parcours manuel de validation (tâche dédiée) le vérifie de bout en bout sur un vrai conflit ; si son jeu d'outils l'interdit, le corriger est un changement de la configuration du worker, hors de celui-ci.
- [`main` avance encore pendant la résolution et l'approbation suivante rencontre un nouveau conflit] → Le parcours est répétable à l'identique ; la borne `target_moving` ne s'applique qu'aux intégrations propres.
- [Réouverture trop zélée : l'utilisateur doit refaire un parcours manuel après une résolution triviale] → La case est modifiable ; elle n'est cochée par défaut que dans le parcours conflit, et reste décochée pour une correction ordinaire.
- [Texte prérempli long dans la tâche de correction quand il y a beaucoup de fichiers] → La liste est citée (`  > `) sous la première ligne par `AppendCorrection` ; elle n'ajoute aucune tâche. Au-delà d'une vingtaine de fichiers, le gabarit tronque la liste avec un décompte.
- [Coche humaine périmée après une intégration propre] → Hors périmètre, documenté dans `docs/opensp8c/workflows.md`.
- [Chemins de fichiers contenant des caractères spéciaux] → `--name-only` peut échapper certains caractères ; on passe `-z` à git et on découpe sur NUL.

## Open Questions

- Faut-il, plus tard, étendre la case de réouverture à la correction de vérification et à l'intégration sans conflit ? Sans effet sur les specs ni les tâches de ce change.
