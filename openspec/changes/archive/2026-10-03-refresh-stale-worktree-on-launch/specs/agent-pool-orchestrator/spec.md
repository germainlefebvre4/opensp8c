## MODIFIED Requirements

### Requirement: Reprise de la branche et du worktree existants
Lorsqu'un worker prend en charge un changement dont la branche `feature/<change>` existe déjà, le backend SHALL réutiliser cette branche et, si le worktree du changement existe déjà, ce worktree tel quel, sans recréer la branche ni écraser les modifications non commitées qu'il contient. Un worktree créé à l'ancien emplacement `.opensp8c/worktrees/wt-<change>` (sans segment de workspace) qui est un worktree enregistré du dépôt du workspace sur cette branche SHALL être réutilisé tel quel. L'existence de la branche SHALL être déterminée par le résultat de git et non par le contenu de sa sortie. Ce comportement s'applique à toute reprise d'un changement : reprise après pause, relance après « Demander des corrections », redémarrage du pool.

Par exception, une branche périmée SHALL être recréée : lorsque le worktree réutilisé ne contient pas `tasks.md` du changement alors que celui-ci est committé dans la branche courante du dépôt, et que la branche `feature/<change>` ne porte aucun travail (worktree sans modification et aucun commit absent de la branche courante), le backend SHALL supprimer ce worktree et cette branche sans forcer, les recréer à partir de la branche courante et ré-enregistrer la branche de base. Une branche qui porte du travail SHALL NOT être supprimée ni réécrite.

#### Scenario: Reprise après pause
- **WHEN** un worker précédemment mis en pause pour le changement `add-user-auth` est repris alors que `feature/add-user-auth` et son worktree existent
- **THEN** le worker réutilise ce worktree avec ses modifications non commitées et n'échoue pas sur l'existence de la branche

#### Scenario: Branche existante sans worktree
- **WHEN** `feature/add-user-auth` existe mais que son worktree a été supprimé
- **THEN** le backend crée un nouveau worktree sur cette branche existante, sans tenter de recréer la branche

#### Scenario: Première prise en charge
- **WHEN** aucune branche `feature/add-user-auth` n'existe
- **THEN** le backend crée la branche et le worktree comme pour un nouveau changement

#### Scenario: Worktree à l'ancien emplacement
- **WHEN** `feature/add-user-auth` est extraite dans un worktree enregistré du dépôt à l'ancien emplacement `.opensp8c/worktrees/wt-add-user-auth`
- **THEN** le backend réutilise ce worktree tel quel avec ses modifications non commitées, sans créer de second worktree ni échouer parce que la branche est déjà extraite

#### Scenario: Branche périmée sans travail recréée
- **WHEN** `feature/add-user-auth` a été créée avant que le changement soit committé, que son worktree est propre sans commit propre, et que le changement est depuis committé dans la branche courante
- **THEN** le backend recrée la branche et le worktree à partir de la branche courante, le worktree contient `openspec/changes/add-user-auth/tasks.md`, et le worker lance l'agent normalement

#### Scenario: Branche périmée avec travail conservée
- **WHEN** `feature/add-user-auth` ne contient pas le changement mais porte des modifications non commitées ou des commits propres
- **THEN** le backend ne supprime ni ne réécrit la branche ni le worktree et le worker passe à `paused` (voir « Présence du change dans le worktree »)

### Requirement: Présence du change dans le worktree
Avant de provisionner la branche et le worktree d'un changement dont la branche `feature/<change>` n'existe pas encore, le worker SHALL vérifier que le fichier `tasks.md` du changement est committé dans la branche courante du dépôt ; sinon il SHALL ne créer ni branche ni worktree, ne lancer aucun agent et passer à `paused` avec une raison de blocage lisible indiquant que le changement doit être committé dans le dépôt avant d'être lancé. Après le provisionnement, le worker SHALL vérifier que `tasks.md` est présent dans le worktree : s'il est absent alors que le changement est committé dans la branche courante, le comportement de recréation ou de pause de la branche périmée décrit dans « Reprise de la branche et du worktree existants » s'applique ; la raison de pause d'une branche périmée portant du travail SHALL se distinguer de celle d'un changement non committé et indiquer que la branche `feature/<change>` ne contient pas le changement et que l'utilisateur doit y intégrer la branche courante ou la supprimer. Si le changement n'est pas committé dans la branche courante, la raison de pause indiquant qu'il doit être committé SHALL être utilisée.

#### Scenario: Change committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont le dossier est committé dans le dépôt
- **THEN** le worktree contient `openspec/changes/add-user-auth/tasks.md` et le worker lance l'agent normalement

#### Scenario: Change non committé
- **WHEN** un worker prend en charge le changement `add-user-auth` dont les fichiers n'ont jamais été committés
- **THEN** aucun agent n'est lancé, le worker passe à `paused` et sa raison de blocage indique que le changement doit être committé avant d'être lancé

#### Scenario: Change non committé sans branche ni worktree orphelins
- **WHEN** un worker prend en charge le changement `add-user-auth` non committé alors qu'aucune branche `feature/add-user-auth` n'existe
- **THEN** le worker passe à `paused` et ni branche `feature/add-user-auth` ni worktree n'ont été créés

#### Scenario: Change committé après un premier échec
- **WHEN** le changement `add-user-auth` a été committé dans la branche courante après une pause pour « non committé », puis le worker est repris
- **THEN** le worker lance l'agent normalement sans que l'utilisateur ait à supprimer la branche ni le worktree

#### Scenario: Branche périmée portant du travail
- **WHEN** `feature/add-user-auth` porte du travail, ne contient pas le changement committé depuis dans la branche courante, et le worker est repris
- **THEN** aucun agent n'est lancé, la branche et le worktree sont intacts, et le worker passe à `paused` avec une raison de blocage distincte de « doit être committé », indiquant que la branche ne contient pas le changement
