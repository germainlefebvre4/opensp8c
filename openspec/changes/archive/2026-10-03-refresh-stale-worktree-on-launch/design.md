# Design

## Context

Dans `runWorker` (`pool/worker.go`), l'ordre est : `Provision` (branche + worktree) → démarrage du journal → vérification de `tasks.md` dans le worktree → subprocess. `Provision` (`pool/worktree.go`) est idempotent par conception : une branche existante est réutilisée, et son worktree n'est jamais remis à zéro pour que le travail non committé survive à une reprise. Un worktree démarre du dernier commit au moment de sa création ; une branche `feature/<change>` ne reçoit ensuite jamais les commits de la branche courante. Voir proposal.md pour le scénario qui a révélé le problème.

Deux helpers existants conviennent : `HasWork(change)` (worktree sale, ou commits de `HEAD..feature/<change>`) et `Remove(change)` (retrait de worktree sans force).

## Goals / Non-Goals

**Goals:**
- Ne jamais laisser de branche ou de worktree orphelins après un échec « change non committé ».
- Sortir automatiquement du cas « branche créée avant le commit du change et jamais utilisée ».
- Garder intacte toute branche qui porte du travail.

**Non-Goals:**
- Rebaser ou fusionner automatiquement la branche courante dans une branche portant du travail : conflits possibles, décision de l'utilisateur.
- Modifier la sémantique de reprise « tel quel » pour un worktree qui contient déjà `tasks.md`.
- Gérer un `tasks.md` committé mais modifié depuis dans la branche courante (le contenu n'est pas comparé, seule la présence l'est).

## Decisions

**1. Garde de commit avant `Provision`, uniquement quand la branche n'existe pas.** Vérifier que `openspec/changes/<change>/tasks.md` est connu de HEAD (`git cat-file -e HEAD:<chemin>`, code de sortie 0 = présent, interprété comme `branchExists` : le code de retour de git décide, pas sa sortie). Absent → pause « doit être committé », rien n'est créé. Quand la branche existe déjà, cette garde ne s'applique pas : la branche peut légitimement porter le change sans que HEAD le contienne encore (travail antérieur), et c'est la vérification dans le worktree qui tranche. *Alternative écartée* : vérifier systématiquement sur HEAD — refuserait à tort une reprise après un changement de branche courante par l'utilisateur.

**2. Branche périmée : trois issues, décidées après `Provision`.** Le worktree ne contient pas `tasks.md` :
```
HEAD contient le change ?
  non -> pause "doit être committé"            (garde inchangée)
  oui -> HasWork(change) ?
           non -> recréer depuis HEAD, relancer la vérification   (aucune perte)
           oui -> pause "la branche ne contient pas le change"    (rien supprimé)
```
La recréation se fait en nouvelle opération `RecreateFromHead(change)` : `Remove` (sans force), puis suppression de la branche avec `git branch -d` (sans force : refuse si la branche n'est pas entièrement contenue dans HEAD, ce que `HasWork == false` garantit déjà), puis `worktree add -b` depuis HEAD et `recordBase` (la base est ré-enregistrée à la création de la nouvelle branche, la précédente disparaît avec l'ancienne). Aucune étape ne force : si l'une échoue, le worker passe en pause avec la raison de provisionnement au lieu de perdre quoi que ce soit. *Alternative écartée* : réutiliser `Discard` (forcé, réservé à l'annulation explicite par l'utilisateur) — son usage ici contredirait la règle « jamais de suppression forcée hors annulation explicite ».

**3. Une seule nouvelle tentative.** Après recréation, la vérification de `tasks.md` est rejouée une fois ; un nouvel échec passe en pause avec le message « doit être committé » (cas où le chemin existe dans HEAD mais pas dans le worktree : situation anormale, signalée plutôt que bouclée).

**4. Raisons de pause.** Deux textes distincts, en français comme les autres raisons de blocage : l'actuel « doit être committé dans le dépôt », et un nouveau « la branche `feature/<change>` ne contient pas le change (créée avant son commit) : y intégrer la branche courante ou la supprimer, puis reprendre le worker ». Aucun changement d'API.

## Risks / Trade-offs

- **[Branche sans travail apparent mais utile : par exemple une branche poussée ailleurs]** → `branch -d` refuse de supprimer une branche non contenue dans HEAD ; ce cas passe par la branche « avec travail » et reste intact.
- **[HEAD détaché]** → `HEAD` désigne alors un commit : la garde et la recréation s'y appliquent (cohérent avec « Aucune base n'est enregistrée lorsque le HEAD est détaché »).
- **[Course : l'utilisateur committe pendant la vérification]** → au pire une pause supplémentaire, résolue par une reprise ; aucune perte de données.
