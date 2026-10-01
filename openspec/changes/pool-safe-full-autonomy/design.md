# Design

## Context

Voir proposal.md pour la motivation. État actuel, dans `backend/internal/pool/` :

- `DetectValidationCommands(root)` (`validation.go`) retourne uniquement les commandes retenues ; un `package.json` avec script `test` mais sans `node_modules` est écarté avec un `log.Printf`. `runValidation` ne retourne un `*ValidationEnvError` que si la liste est vide.
- `Provision` (`worktree.go`) crée la branche par `git worktree add -b feature/<change> <path>` sans point de départ explicite : elle part du HEAD du dépôt à cet instant. Rien ne retient d'où elle part.
- `MergeInto` fusionne dans `CurrentBranch()` évaluée à l'instant du merge, sous `m.mergeMu`.
- `runWorker` (`worker.go`) enchaîne : tour d'agent → `runValidation` → boucle de guérison → contrôle `tasks.md` → `CommitAll` → `HasWork` → (full-autonomy) `mergeMu.Lock` → `MergeInto` → `Remove` → `DeleteBranch`. Validation et guérison sont du code en ligne : elles ne peuvent pas être rejouées après coup.
- Le subprocess de l'agent reste vivant jusqu'à la sortie de `runWorker` (même session pour l'apply et chaque heal) : on peut donc lui renvoyer un tour de guérison après une intégration.
- Le change `pool-cancel-before-merge` (en cours, indépendant) ajoute des contrôles `ctx.Err()` avant `CommitAll` et juste après `mergeMu.Lock()`, et modifie l'exigence « Fusion sûre en mode full-autonomy ». Ce change-ci ne la modifie pas.

## Goals / Non-Goals

**Goals:**
- Aucun projet de test détecté ne peut être sauté sans que le worker le dise.
- Le travail d'un agent n'est fusionné que dans la branche dont il est issu, et uniquement sous la forme déjà validée.
- Le comportement multi-workers existant (fusions sérialisées, aucun échec dû à un autre worker) est conservé.

**Non-Goals:**
- Installer automatiquement les dépendances Node dans le worktree.
- Valider autre chose que ce que la commande de validation exécute (typecheck, lint, build restent à la charge de la commande configurée).
- Sécuriser ou isoler les écritures de l'agent hors du worktree.
- Rendre `git merge` annulable ou traiter l'annulation (voir `pool-cancel-before-merge`).
- Protéger le working tree vivant de l'utilisateur contre un merge qui chevauche ses modifications non committées : git refuse déjà ce merge, et le worker se met en pause avec la cause.

## Decisions

### 1. Détection : un projet non validable devient une erreur d'environnement

`DetectValidationCommands` est remplacée en interne par une fonction qui retourne les commandes **et** la liste des répertoires détectés mais non validables (chemins relatifs à la racine du worktree). `runValidation`, en auto-détection, retourne un `*ValidationEnvError` dès que cette liste n'est pas vide, **avant** d'exécuter la moindre commande : pas de 20 minutes de `go test` gaspillées pour finir en pause. La raison nomme les répertoires et propose une commande type (`npm ci && npm test`). `DetectValidationCommands` est conservée avec sa signature actuelle (elle délègue) pour ne pas casser ses tests ni ses éventuels appelants.

L'erreur d'environnement existante fait déjà le reste : pause immédiate, aucune tentative consommée, raison exposée par le statut du pool et visible dans l'UI.

*Alternatives écartées :*
- **Installer les dépendances (`npm ci`) automatiquement** : réseau, durée, effets de bord et surface d'attaque (scripts d'install d'un dépôt arbitraire) qui n'ont pas à être décidés par la plateforme ; l'utilisateur le fait explicitement dans sa commande de validation.
- **Lier `node_modules` du dépôt principal dans le worktree** : état mutable partagé entre l'agent et le dépôt de l'utilisateur, et l'installation peut diverger de `package-lock.json` du worktree.
- **Ne pauser qu'en full-autonomy et se contenter d'un avertissement en `hitl-review`** : deux comportements de détection selon le mode, et l'avertissement est exactement le « log silencieux » qu'on retire. La validation sert aussi la revue : on la veut fiable partout.
- **Ne pauser que si aucune autre commande n'est détectée** : c'est l'état actuel du défaut.

### 2. Branche de base : enregistrée dans la configuration git du dépôt

À la création de la branche (chemin « première fois » de `Provision`), après `worktree add -b`, le backend exécute `git config branch.feature/<change>.opensp8c-base <branche courante>`, sauf si le HEAD est détaché (rien n'est enregistré). Lecture : `git config --get` sur la même clé ; code 1 = pas de base.

Pourquoi la configuration git :
- Elle vit dans le dépôt, donc survit à un redémarrage du serveur et à la perte d'état en mémoire du `Manager`, sans introduire de fichier d'état de plus.
- Git supprime la section `branch.<nom>.*` quand la branche est supprimée (`branch -d`/`-D`), ce qui donne gratuitement le cycle de vie voulu.
- Elle est locale (non poussée) et n'ajoute aucun fichier versionné.

Elle n'est jamais réécrite lors d'une reprise (chemins « branche existante »), ce qui fait qu'un utilisateur qui change de branche pendant une pause ne déplace pas la base. L'écriture est « au mieux » : un échec de `git config` journalise une alerte et laisse la branche sans base (comportement d'avant), plutôt que de bloquer le provisionnement.

*Alternatives écartées :*
- **Stocker la base dans `preferences.json` ou un fichier d'état** : état de plus à garder cohérent avec les branches, et `preferences.json` est un fichier utilisateur partagé entre tous les workspaces.
- **Stocker la base en mémoire dans `Worker`** : perdue au redémarrage, au moment même où une reprise après pause est le cas d'usage.
- **Déduire la base par `git merge-base` ou le reflog** : ne donne pas un nom de branche, et le reflog est local, expirable et ambigu.

### 3. Contrôle de branche de base : à l'entrée de la finalisation full-autonomy, et dans `MergeInto`

Le contrôle est fait deux fois, pour deux raisons différentes :
- **Tôt**, juste avant l'étape d'intégration (décision 4) : c'est ce qui évite d'intégrer et de revalider (potentiellement des minutes) pour un merge qui sera de toute façon refusé.
- **Dans `MergeInto`**, à l'instant du merge : une erreur typée `ErrBaseBranchMismatch` (avec les deux noms) est retournée avant tout `git merge`. C'est ce second contrôle qui ferme la fenêtre entre la vérification et le merge et qui ne dépend pas de l'ordre des étapes du worker. `worker.go` mappe l'erreur sur une pause lisible, comme pour `ErrMergeInProgress`.

`MergeInto` est le bon endroit pour le second contrôle : il est le seul chemin vers `git merge`, et `pool-cancel-before-merge` déclare ne pas le modifier, ce qui limite le conflit entre les deux changes à quelques lignes du bloc d'erreurs de `worker.go`.

Sans base enregistrée, aucun des deux contrôles ne s'applique (compatibilité des branches existantes).

### 4. Intégration avant le verrou, revalidation par la boucle existante

Nouvelle étape entre `HasWork` et `mergeMu.Lock()` en full-autonomy :

```
CommitAll -> HasWork -> [controle base] -> [cible avance ?]
                                              |non                |oui
                                              v                    v
                                         mergeMu.Lock      merge <cible> dans le worktree
                                                              |conflit -> abort + pause
                                                              v
                                                       rejouer validation + guerison
                                                              |echec -> pause
                                                              v
                                                       CommitAll (corrections)
                                                              v
                                                         mergeMu.Lock
                                                              v
                                          [cible contient-elle un commit absent ?] -> oui: pause
                                                              v
                                                          MergeInto
```

Test « la cible a-t-elle avancé » : `git merge-base --is-ancestor <cible> feature/<change>` (code 0 = rien à intégrer, 1 = commits manquants). Ce test n'a besoin d'aucun SHA enregistré, il reste vrai après une intégration manuelle de l'utilisateur pendant une pause, et il vaut pour la branche cible réelle (celle qu'on s'apprête à fusionner), y compris pour une branche sans base enregistrée.

Intégration : `git merge --no-edit <cible>` exécuté dans le worktree (propre après `CommitAll`). En cas d'échec, `git merge --abort` dans le worktree, puis pause avec la sortie de git tronquée (même `truncateReason` que les autres pauses). Branche et worktree sont dans l'état d'avant l'intégration.

Revalidation : l'actuel enchaînement « `runValidation` → boucle de guérison » du worker est extrait en une fonction locale à `runWorker` qui partage le compteur `attempts`, de sorte que le budget `max_attempts` couvre la validation initiale **et** la validation après intégration (un agent ne dispose pas d'un budget neuf à chaque revalidation). Cette fonction renvoie « ok » ou la cause de la pause, avec les mêmes `pause(...)` qu'aujourd'hui, et les messages de pause existants sont conservés tels quels. L'extraction est un refactoring sans changement de comportement pour le chemin existant, protégé par les tests actuels (`worker_test.go`, `finalize_test.go`), qui doivent passer inchangés avant d'ajouter l'étape d'intégration.

Après le verrou, un dernier test d'ascendance (dans le worker, avant `MergeInto`) couvre l'avancée de la cible pendant l'intégration. Si elle a avancé, pause explicite plutôt qu'une boucle d'intégration à l'intérieur du verrou : l'utilisateur qui commite pile pendant la fenêtre est rare, et tenir `mergeMu` pendant une validation de 20 minutes bloquerait les autres workers.

*Alternatives écartées :*
- **Refuser la fusion dès que la cible a avancé** : simple, mais contredit l'exigence existante « Deux workers qui finalisent en même temps » (le second worker verrait toujours la cible avancée par le premier et se mettrait en pause à chaque fois).
- **Intégrer et revalider sous `mergeMu`** : correct par construction (la cible ne peut plus bouger du fait d'un autre worker) mais sérialise les validations entières, soit jusqu'à 20 minutes de blocage par worker.
- **Fusionner puis valider sur la cible, avec `reset` en cas d'échec** : modifie la branche et le working tree de l'utilisateur avant de savoir si le résultat est bon, et un reset est précisément ce que la sûreté du merge a retiré.
- **Se contenter d'un avertissement** : ne protège pas le cas visé (merge non testé dans la branche de l'utilisateur).

### 5. Stratégie de test

Tous les tests utilisent des dépôts git temporaires et le seam `startSubprocessFn` existant (`pipeAgent`), sans réseau :
- Détection : couvre les trois cas (sans projet, avec projet non validable seul, avec projet non validable + module Go) et vérifie qu'aucune commande n'a été lancée (script de validation qui écrit un fichier témoin).
- Base : `Provision` enregistre la base à la création, ne la réécrit pas à la reprise, rien en HEAD détaché ; la suppression de la branche supprime la clé.
- Contrôle de base : `MergeInto` renvoie `ErrBaseBranchMismatch` sans lancer `git merge` (HEAD de la cible inchangé) ; worker : pause avec les deux noms, branche et worktree conservés ; retour sur la base puis reprise : fusion réussie ; branche sans base : comportement inchangé.
- Intégration : cible inchangée (aucune revalidation, vérifiée par un compteur d'exécutions de validation) ; cible avancée sans conflit (une revalidation, fusion après) ; échec de validation après intégration puis guérison réussie ; conflit (état du worktree identique à avant, `MERGE_HEAD` absent) ; cible avancée pendant la fenêtre (commit injecté via la validation rejouée) ; deux workers où le second revalide après le premier (réutilise le montage de `TestFinalize_ConcurrentMergesAreSerialized`).
- `go test ./internal/pool -race` doit passer, ainsi que `go test ./... -race` depuis `backend/`.

## Risks / Trade-offs

- [Les dépôts Node sans commande de validation configurée se mettent désormais en pause au premier run] → c'est le but ; la raison indique la commande à configurer, et c'est le seul comportement changé pour les workspaces non configurés.
- [Une intégration réussie peut produire un résultat qui échoue à la validation et consomme des tentatives de guérison] → budget `max_attempts` partagé avec la validation initiale ; au pire le worker se met en pause avec la branche conservée.
- [Une branche sans base enregistrée n'est pas protégée] → vrai pour les branches créées avant ce change ; elles se terminent normalement. Aucune migration automatique : deviner la base d'une branche existante serait pire que ne pas la contrôler.
- [`git config` est partagé entre tous les worktrees du dépôt] → c'est voulu : la clé est propre à une branche et le même dépôt gère tous ses worktrees ; deux workspaces ayant un change de même nom ont des dépôts différents.
- [Un utilisateur qui change volontairement de branche pendant le run voit une pause] → la raison nomme les deux branches ; revenir sur la base et reprendre suffit. C'est le choix prudent pour un merge automatique.
- [Conflit de fusion textuelle avec `pool-cancel-before-merge` dans le bloc full-autonomy de `worker.go`] → les deux changes insèrent des contrôles à des endroits distincts (avant `CommitAll` et après le verrou pour l'autre ; avant le verrou et dans le bloc d'erreurs pour celui-ci). Appliquer l'un puis rebaser l'autre ; aucun conflit de spec puisque les exigences touchées sont distinctes.

## Migration Plan

Aucune donnée persistée par l'application ne change ; la seule écriture nouvelle est une clé de configuration git locale, ignorée par les versions précédentes. Déploiement en une fois côté backend. Retour arrière : revert du commit ; les clés `opensp8c-base` déjà écrites restent inertes.

## Open Questions

- L'intégration doit-elle utiliser `merge` (adopté ici : conserve l'historique de la branche, jamais de réécriture) ou `rebase` (historique linéaire, mais réécrit des commits que l'agent et l'utilisateur ont pu observer) ? Le choix n'affecte ni les specs ni les tâches ; on part sur `merge`.
