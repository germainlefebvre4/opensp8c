# Design

## Context

Voir `proposal.md` (Why). État actuel observé, après la livraison de `resume-paused-worker-after-manual-tasks` et de la validation humaine (`human-review-tasks`) :

- `runWorker` enchaîne : provisionnement, subprocess, `/opsx:apply`, triage (hitl), validation avec guérison, contrôle de complétude (`tasksBlockCompletion`), commit, `HasWork`, puis la finalisation (`integrateAndMerge` en `full-autonomy`, `MarkReview` en `hitl-review`). Un worker `finalizeOnly` saute le subprocess et reprend à la validation.
- Un worker en pause n'existe qu'en mémoire ; `finalizeChanges` (drapeau en mémoire) transmet « reprendre en finalisant » au nouveau worker créé par `tick`, qui reste l'unique point de dispatch.
- `GetRunnableChanges` ne propose que les changes `todo` ; `tick` sort tôt quand tous les slots de workers sont pris.
- Le statut `to-review` est un marqueur persistant : clé `branch.feature/<change>.opensp8c-review` de la configuration git du dépôt, lue en un appel par `ReviewMarkers`, posée par `MarkReview`, levée par `ClearReview`, supprimée avec la branche. `ListChanges` lui donne priorité (`markReviewed`). Les tâches « suivent la branche » : tant qu'aucun worker ne tient le change, le détail et les compteurs lisent le `tasks.md` de la branche, et une coche crée un commit sous le verrou `reviewLock`.
- Les runs sont journalisés par `openRunLog` / `logRun` (type `pool`), sur un `Worker` ; `runTurn` ne renvoie qu'une erreur, pas le texte du résultat.
- Le slot Done / Archived de `KanbanPage` empile déjà deux `KanbanColumn` (flex-1 et `max-h-[40%]`). Le slot In Progress est une `KanbanColumn` du tableau `leadingColumns`.
- Aucun rôle n'est lancé pour une vérification, et la valeur de configuration de vérification n'existe pas encore : elles viennent de `verification-settings` (prérequis), qui expose la résolution d'une valeur `{conformity, ui}` pour un workspace et un change, et le rôle `verifier`.

## Goals / Non-Goals

**Goals:**
- Une étape persistante qui libère les slots du pool et survit au redémarrage, sans nouvelle sémantique de pause.
- Réutiliser les mécanismes existants : marqueur git, `tick`, `finalizeOnly`, run journalisé, verrou d'actions du change.
- Une structure d'étapes ordonnées, pour que `ui-verify-step` ajoute la sienne (avec son verrou exclusif) sans toucher à l'exécuteur.

**Non-Goals:**
- Vérification UI, verrou exclusif, lancement de l'application.
- Réparation automatique, relance automatique, reprise d'une vérification à mi-parcours après un redémarrage (une exécution interrompue est recommencée, ce qui coûte à nouveau des tokens).
- Badge `verifying` dans le sidebar, affichage des vérifications dans la modal du pool.

## Decisions

### D1. Un marqueur git à trois états, calqué sur le marqueur de revue

Clé `branch.feature/<change>.opensp8c-verify`, valeurs `pending` | `failed` | `passed`. `openspec.VerifyMarkers(workspacePath) map[string]string` (un seul `git config --get-regexp`, jamais d'erreur propagée, comme `ReviewMarkers`) ; `WorktreeController.SetVerify(change, state)`, `ClearVerify(change)`, `VerifyState(change)` ; une valeur inconnue est lue comme `failed` (fail-safe : jamais de relance ni de finalisation implicites).

`ListChanges` applique, après le marqueur de revue : `if markers[ch.Name] != "" && ch.HasBranch && ch.KanbanStatus != "to-review"` → `markVerifying(ch, state)` qui pose `KanbanStatus = "verifying"`, `IsStale = false` et `VerificationState` (`pending` → `queued`, `failed`, `passed`). La revue passe donc avant la vérification (un marqueur de revue résiduel ne cache pas un change à relire). Le `Change` gagne `VerificationState` et `VerificationStep` (`json:",omitempty"`) ; le handler remplace `queued` par `running` (et renseigne l'étape) à partir de l'état en mémoire du `Manager`, comme `worker_active` remplace l'absence d'information sur le worker.

*Alternatives :* (a) fichier dans le worktree : serait commité par `CommitAll` ; (b) champ de `.openspec.yaml` : versionné, voyagerait dans la fusion et mélangerait état d'exécution et métadonnées de change ; (c) état en mémoire seul : perdu au redémarrage, ce qui ferait perdre aussi la file et obligerait à relancer des vérifications payantes.

### D2. Le worker diverge juste avant la finalisation

Dans `runWorker`, entre `HasWork` et l'étape 8 :

```go
if !w.finalizeOnly && m.verificationRequired(w) {
    wt.SetVerify(change, "pending") ; publish change_updated ; outcome = OutcomeAwaitingVerification ; return
}
```

`verificationRequired` résout la valeur (Configuration, workspace, puis `.openspec.yaml` lu dans le dépôt principal, voir `verification-settings`) et renvoie vrai si au moins une étape est activée. Un seul adaptateur, `Manager.resolveVerification(workspaceID, change)`, isole cette dépendance : les noms exacts exposés par `verification-settings` ne sont référencés que là. Le worker rend son slot par le chemin de sortie normal (`defer` de `runWorker`) ; aucune pause n'est créée, donc ni `pausedWorkers` ni badge de pause. Le worktree et la branche restent : ni `Cleanup` ni `Discard` ne sont appelés. Une constante `OutcomeAwaitingVerification = "awaiting-verification"` rejoint les issues de `runlog.go`.

Un worker `finalizeOnly` ne passe jamais par ce point : c'est ce qui évite la boucle « vérification → finalisation → vérification », et c'est le comportement voulu pour « Reprendre en finalisant » après une pause.

### D3. L'exécuteur vit dans le `Manager`, piloté par le même `tick`

`Manager` gagne `verifications map[string]*verification` (en cours, gardé par `m.mu`). `tick` est scindé : `tickWorkers(changes)` (logique actuelle, y compris la sortie anticipée quand les slots sont pleins) puis `tickVerifications(changes)`, qui reçoivent la même liste issue d'un unique `ListChanges`. Ainsi des slots de workers pleins n'empêchent plus de traiter les vérifications, et inversement.

`tickVerifications` :
1. pour chaque change `verifying` en état `queued` et non présent dans `m.verifications`, par ordre de nom, tant que `len(m.verifications) < m.config.Size` : `startVerification`.
2. pour chaque change `verifying` en état `passed` qu'aucun worker ne tient, tant qu'un slot de worker est libre : `startWorker(change)` avec `finalizeOnly`.

`startVerification` enregistre l'entrée, `m.workers.Add(1)` (pour que `StopAndWait` attende aussi les vérifications), et lance `runVerification` dans une goroutine avec un `context` annulable stocké dans l'entrée. `Stop` annule toutes les entrées et vide la map ; le marqueur reste `pending` (voir spec). La limite de concurrence prend la taille du pool comme valeur plutôt qu'un réglage dédié : un réglage de plus ne se justifie pas tant que la vérification est désactivée par défaut.

*Alternative :* une goroutine de vérification par change démarrée par le worker lui-même, après avoir libéré le slot. Écartée : elle ne survit pas au redémarrage, et elle dupliquerait la file que le `tick` fournit déjà.

### D4. `runVerification` : étapes ordonnées, conformité en première

```go
type verifyStep interface {
    Name() string                                  // "conformity"
    Enabled(verification.Resolved) bool
    Run(ctx context.Context, v *verifyRun) stepResult // {Verdict, Reason, Report}
}
var verifySteps = []verifyStep{conformityStep{}}
```

`runVerification` : ouvre le journal (`verify`), pose l'entrée d'activité `pool.verification_started`, résout la configuration, parcourt les étapes activées dans l'ordre et s'arrête à la première qui n'a pas réussi. Aucune étape activée ou toutes réussies → `SetVerify(change, "passed")`. Sinon → `SetVerify(change, "failed")`. Dans tous les cas : marqueur de fin du journal, entrée d'activité, `change_updated`, retrait de l'entrée de `m.verifications`. Une annulation (`ctx.Err() != nil`) ne modifie pas le marqueur.

`conformityStep.Run` :
- worktree : `wt.Provision(change)` (recrée le worktree s'il a disparu, comme `RequestCorrection`) ;
- subprocess : `startSubprocessFn(…, agentCfg de rôle verifier, extraPrompt = verifyDirective, …)` ; la directive interdit toute modification de fichier et impose la ligne finale `VERDICT: PASS|FAIL` (FAIL dès qu'un point critique est relevé) ;
- un seul tour `/opsx:verify <change>` ;
- verdict : dernière ligne du texte du résultat qui correspond à `^VERDICT:\s*(PASS|FAIL)\s*$`, insensible à la casse ; absence = échec « verdict absent » (fail-closed) ;
- contrôle d'intégrité : `git status --porcelain` du worktree avant et après ; toute différence est un échec « le vérificateur a modifié le worktree », fichiers laissés pour examen ;
- démarrage, tour en erreur et inactivité (`agentIdleTimeout`) sont des échecs avec leur raison.

Obtenir le texte du résultat impose de généraliser `runTurn`. Il est réécrit en `runTurnText(t turnTarget, proc, content) (string, error)`, où `turnTarget` regroupe ce dont la boucle de lecture a besoin (`logRun`, `setActivity`, `notify`, `procCancel`) ; `Worker.runTurn` en devient un adaptateur qui ignore le texte, sans changement de comportement. Le texte vient du champ `result` de l'événement final ; à défaut (traductions d'autres agents), des deltas de texte accumulés pendant le tour. Si aucun texte n'est obtenu, le verdict est absent donc l'étape échoue.

*Alternatives :* (a) verdict structuré par un fichier écrit par l'agent : exigerait d'autoriser une écriture, contredisant la lecture seule ; (b) interpréter le rapport Markdown du skill (comptes de CRITICAL / WARNING) : fragile face à ses variations de format, alors qu'une ligne de verdict explicite se vérifie par une regex.

### D5. Réussite : le marqueur `passed` porte l'intention de finaliser

Plutôt qu'un second drapeau en mémoire, `passed` est l'intention : `tickVerifications` voit le change `verifying`/`passed` et appelle `startWorker`, qui reçoit `finalizeOnly = true` quand `m.finalizeChanges[change]` **ou** que le marqueur vaut `passed`, et lève le marqueur (`ClearVerify`) à ce moment, avant de lancer la goroutine. Le change quitte alors `verifying` : sa carte passe dans In Progress avec le badge du worker ; son statut vient du worktree (`ApplyWorktreeProgress` plafonne à In Progress). Cette intention survit donc au redémarrage et il n'y a pas de fenêtre où le change retomberait dans To Do.

Si le worker de finalisation se met en pause (validation qui échoue, complétude insuffisante), le marqueur est déjà levé : c'est le fonctionnement existant d'une pause, avec « Reprendre » et « Reprendre en finalisant ». Un « Reprendre » ordinaire relancerait un agent puis repasserait par la vérification ; c'est cohérent (le code a changé) et visible pour l'utilisateur.

Les dépendances ne sont pas revérifiées pour un `passed` : elles l'étaient au dispatch du worker d'origine.

### D6. Journal et rapport : un run `verify` sans `Worker`

`openRunLog`, `logRun`, `logRunMarker`, `throttleFor`, `noteRunAppended` et `finishRunEvents` dépendent de `*Worker` uniquement pour `WorkspaceID`, `ActiveChange`, `ID` (journal) et `runLog`. Ils sont refactorés sur une petite structure `runRef{workspaceID, change string; id int; log *poolRunLog}` que `Worker` et `verifyRun` fournissent ; `Worker` conserve ses méthodes comme enveloppes, de sorte que les tests existants passent sans modification. Le run de vérification utilise le type de conversation `verify` et un identifiant horodaté (`runTimestampLayout`).

Le marqueur de fin `verify_run_end` porte `{step, verdict, reason, report}`. `GET …/verification/report` lit le dernier run `verify` du change (`convStore.List` puis `Load`, comme `pool_runs.go`) et en extrait ce marqueur ; il ne maintient aucun état propre. L'onglet Log du DetailPanel liste déjà les runs par type : il doit seulement connaître `verify`.

### D7. Actions après échec : verrou du change, correction partagée

Un `verificationHandler` expose `rerun`, `finalize`, `request-correction` et `report`. Les trois actions prennent `TryLockReview(change)` (même verrou que les actions de revue, `review_busy` si pris) et vérifient dans cet ordre : change et workspace connus (`404`), aucune vérification en cours (`verification_running`), marqueur `failed` (`not_failed`).

- `rerun` : `SetVerify(change, "pending")`.
- `finalize` : lit le `tasks.md` de la branche (`BranchTasks`) ; s'il reste une tâche non cochée → `409 tasks_incomplete` avec `remaining` (règle identique à « Reprendre en finalisant » : toutes les tâches cochées, y compris celles marquées validation humaine) ; sinon `SetVerify(change, "passed")`.
- `request-correction` : le corps de `RequestCorrection` (ajout à « ## Corrections », commit, levée du marqueur) est extrait en une fonction `applyCorrection(wt, change, feedback, clear func() error)` que les deux chemins appellent ; seul le marqueur levé diffère (`ClearReview` / `ClearVerify`) et le contrôle d'éligibilité (`checkReviewable` / `checkFailed`).

Ces actions publient `change_updated`. Le pool n'a pas besoin de tourner pour qu'elles réussissent ; `rerun` ne s'exécute toutefois que pool démarré, et l'interface l'indique par l'état `queued`.

*Alternative :* réutiliser `…/review/request-correction`, en élargissant `checkReviewable` à `failed`. Écartée : cela modifierait la spec `change-review-actions` pour un contrat qui n'a pas à changer, et mélangerait deux machines à états.

### D8. Colonne : ce que le front calcule

`KanbanPage` remplace l'entrée `in-progress` de `leadingColumns` par un slot composite, sur le modèle Done / Archived :

```
<div class="flex-1 min-w-[220px] flex flex-col min-h-0 gap-2">
  <KanbanColumn status="in-progress" className="flex-1 min-h-0" .../>
  {showVerifying && <><div class="h-px bg-slate-200 shrink-0"/>
  <KanbanColumn status="verifying" className="max-h-[40%] overflow-y-auto" .../></>}
</div>
```

`showVerifying = verifyingChanges.length > 0 || workspaceVerificationEnabled`, la seconde valeur venant de `resolved.verification` de `useWorkspaceSettings(workspaceId)` (déjà en cache pour l'écran Settings), `conformity || ui`. `VALID_DROPS` ne cite pas `verifying` : ni source, ni cible. Dans `ChangeCard`, le drag est désactivé pour `verifying` (même mécanisme que pour `done`). Les badges d'état suivent le motif des badges de worker. Les boutons « Relancer » / « Finaliser » partagent un hook `useVerificationActions(workspaceId)` (mutations, erreur du backend, invalidation de `['changes', id]` et `['change-detail', id, name]`), utilisé aussi par le bandeau du DetailPanel, comme `useResumeWorker`.

### D9. Interactions avec les autres actions

- `task.go` : `PatchTask` consulte l'état en mémoire (`Manager.VerificationRunning(change)`) et répond `409 verification_busy` si une vérification tourne ; en `pending` sans exécution, `failed` ou `passed`, la coche suit le comportement de la branche (aucun worker ne tient le change).
- Réinitialisation vers To Explore (`ff.go`) : appelle aussi `ClearVerify` à côté de la levée du marqueur de revue ; refusée en `verification_busy` tant qu'une vérification tourne (même emplacement que `worker_active`). La suppression de la branche supprime de toute façon la section git du marqueur.
- Compteurs (`workspace.go`) : la clé `verifying` est ajoutée au dictionnaire initial.

## Risks / Trade-offs

- [Une vérification interrompue par l'arrêt du pool ou un redémarrage est relancée depuis le début, donc des tokens dépensés deux fois] → assumé : la consommation est sous contrôle de l'interrupteur de configuration, et reprendre à mi-parcours exigerait de persister le contexte de l'agent.
- [La validation locale (tests) tourne deux fois : avant la vérification et dans le worker de finalisation] → acceptée : le worker de finalisation réutilise `finalizeOnly` tel quel, et un second passage protège contre une dérive du worktree pendant l'attente.
- [Le vérificateur écrit malgré la directive] → contrôle `git status` avant / après ; l'échec nomme les fichiers et les laisse en place pour examen.
- [Faux échecs : verdict mal formé ou agent bavard] → fail-closed assumé ; la relance est un clic, et le rapport est conservé pour comprendre. Pas de boucle de réparation : le coût d'un faux positif reste borné à une exécution.
- [`verifying` est un nouveau statut : tout code qui énumère les statuts peut l'oublier] → tâche dédiée qui recense `grep "to-review"` côté Go (`scheduler.go`, `workspace.go`, `change.go`) et TypeScript (`KanbanColumn` styles, `VALID_DROPS`, filtres de recherche, sidebar), avec un test par site.
- [Dépendance aux noms exposés par `verification-settings`, pas encore appliqué] → un unique adaptateur (`resolveVerification`) ; les tâches de ce change commencent par vérifier que `verification-settings` est appliqué.
- [`tick` détient `m.mu` pendant `ListChanges` (appels git)] → comportement existant, inchangé ; la liste est lue une fois par tick pour les deux dispatchs.
- [Plusieurs workers de finalisation en attente de slot pendant que d'autres tournent] → `passed` reste visible dans la colonne avec son badge ; le tick sert les slots libres dans l'ordre de nom.

## Migration Plan

Aucune migration : le marqueur est absent des dépôts existants, et la valeur intégrée de `verification-settings` désactive les deux étapes, donc aucun worker ne diverge tant qu'on n'active rien. Retour arrière : désactiver les étapes dans la configuration ; les changes encore marqués peuvent être finalisés par « Finaliser sans vérification », ou le marqueur supprimé par `git config --unset`.

## Open Questions

- Limite de concurrence des vérifications : la taille du pool est un choix par défaut. Elle pourra devenir un réglage si la pratique montre un goulot.
