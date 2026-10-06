# Spec Delta

## Purpose

Définit le format Conventional Commits des commits que l'application crée pour un change (travail du worker, correction de revue, fusion), afin que l'historique du dépôt dise ce que chaque change apporte et où.

## ADDED Requirements

### Requirement: Format Conventional Commits des commits d'un change
Tout commit créé par l'application pour un change SHALL avoir un message de la forme `type(scope): Sujet`, suivi d'une ligne vide puis d'un corps contenant au moins la ligne `Change: <nom-du-change>` (nom kebab-case du change, tel qu'il figure dans `openspec/changes/`). Lorsque le scope est omis, l'en-tête SHALL être `type: Sujet`. Le sujet SHALL être le nom du change formaté : tirets et soulignés remplacés par des espaces, première lettre en majuscule, le reste inchangé. L'en-tête SHALL NE PAS dépasser 72 caractères : au-delà, le sujet SHALL être tronqué sur une frontière de mot, le nom complet restant lisible dans le corps. Le message SHALL NE PAS dépendre de l'état des tâches du change ni de la langue de l'interface.

#### Scenario: Change au nom court
- **WHEN** l'application committe le travail du change `add-user-auth` dont le scope déduit est `auth`
- **THEN** l'en-tête du message est `feat(auth): Add user auth` et le corps contient la ligne `Change: add-user-auth`

#### Scenario: Change au nom long
- **WHEN** le change `improve-matrix-change-drilldown-nav` est committé avec le scope `timeline-spec-matrix`
- **THEN** l'en-tête est `feat(timeline-spec-matrix): Improve matrix change drilldown nav` et le corps contient `Change: improve-matrix-change-drilldown-nav`

#### Scenario: En-tête trop long
- **WHEN** le nom formaté ferait dépasser 72 caractères à l'en-tête
- **THEN** le sujet est tronqué sur une frontière de mot pour que l'en-tête ne dépasse pas 72 caractères, et le corps contient le nom complet du change

#### Scenario: Scope omis
- **WHEN** aucun scope ne peut être déduit pour le change `fix-login-redirect`
- **THEN** l'en-tête est `fix: Fix login redirect` et le corps contient `Change: fix-login-redirect`

### Requirement: Type déduit du nom du change
Le type du message SHALL être déduit du premier mot du nom du change, sans tenir compte de la casse : `fix` donne `fix`, `refactor` donne `refactor`, `perf` donne `perf`, `docs` ou `doc` donne `docs`, `test` ou `tests` donne `test`, `chore` donne `chore`, `ci` donne `ci`, `build` donne `build`, `style` donne `style` ; tout autre premier mot (`add`, `improve`, `create`…) donne `feat`. Le commit de correction de revue SHALL toujours avoir le type `chore`, quel que soit le nom du change.

#### Scenario: Premier mot reconnu
- **WHEN** le change `fix-explore-ghost-card-bugs` est committé
- **THEN** le type du message est `fix`

#### Scenario: Premier mot non reconnu
- **WHEN** le change `improve-matrix-change-drilldown-nav` est committé
- **THEN** le type du message est `feat`

#### Scenario: Correction de revue
- **WHEN** une demande de correction est enregistrée pour le change `fix-login-redirect`
- **THEN** le commit de la correction a le type `chore`

### Requirement: Scope déduit des métadonnées du change
Le scope SHALL être, dans cet ordre : le premier élément non vide de `tags.components` du `.openspec.yaml` du change ; à défaut, le premier dossier de `specs/` du change, par ordre alphabétique ; à défaut, aucun scope. Les métadonnées SHALL être lues dans le dossier du change du dépôt principal, à défaut dans celui du worktree du change. Le scope SHALL être normalisé : minuscules, tout caractère hors `a-z`, `0-9`, `.`, `_`, `/` et `-` remplacé par `-`, tirets consécutifs fusionnés, tirets de début et de fin retirés ; un scope vide après normalisation SHALL être traité comme absent et la source suivante essayée. Un `.openspec.yaml` absent ou illisible, ou un dossier de change introuvable, SHALL NE PAS faire échouer le commit : le scope est alors omis.

#### Scenario: Scope issu des tags
- **WHEN** le `.openspec.yaml` du change porte `tags.components` valant `[timeline-spec-matrix, timeline-page]`
- **THEN** le scope est `timeline-spec-matrix`

#### Scenario: Scope issu des capacités du change
- **WHEN** le change n'a pas de tags mais un dossier `specs/` contenant `task-toggle/` et `kanban-board/`
- **THEN** le scope est `kanban-board`

#### Scenario: Normalisation
- **WHEN** le premier composant est `Agent Pool`
- **THEN** le scope est `agent-pool`

#### Scenario: Métadonnées absentes
- **WHEN** le change n'a ni `.openspec.yaml` exploitable ni dossier `specs/`
- **THEN** le message est produit sans scope et le commit aboutit

### Requirement: Messages des commits du worker, de correction et de fusion
Le commit qui enregistre le travail d'un worker dans `feature/<change>` SHALL avoir le message `type(scope): Sujet` du change, avec le corps `Change: <nom>`. Le commit de correction de revue SHALL avoir l'en-tête `chore(scope): Add review correction` et le corps `Change: <nom>`. La fusion `--no-ff` de `feature/<change>` dans la branche cible (approbation d'une revue ou fusion en `full-autonomy`) SHALL utiliser pour son message le même en-tête et le même corps que le commit du worker. Le message du merge d'intégration de la branche cible dans la branche du change, produit par git, SHALL rester inchangé. Un échec de lecture des métadonnées SHALL NE PAS empêcher ces commits.

#### Scenario: Commit du travail du worker
- **WHEN** le worker du change `improve-matrix-change-drilldown-nav` (scope `timeline-spec-matrix`) termine et que son travail est committé
- **THEN** le commit a pour message `feat(timeline-spec-matrix): Improve matrix change drilldown nav` et un corps `Change: improve-matrix-change-drilldown-nav`

#### Scenario: Commit de correction
- **WHEN** l'utilisateur demande une correction pour ce change
- **THEN** le commit ajouté à `feature/improve-matrix-change-drilldown-nav` a pour en-tête `chore(timeline-spec-matrix): Add review correction`

#### Scenario: Fusion après approbation
- **WHEN** l'utilisateur approuve ce change en revue
- **THEN** le commit de fusion dans la branche cible a pour en-tête `feat(timeline-spec-matrix): Improve matrix change drilldown nav` et pour corps `Change: improve-matrix-change-drilldown-nav`

#### Scenario: Fusion en full-autonomy
- **WHEN** un worker en `full-autonomy` fusionne son change
- **THEN** le commit de fusion porte le même message que lors d'une approbation

#### Scenario: Merge d'intégration inchangé
- **WHEN** la branche cible est intégrée dans la branche du change avant la fusion
- **THEN** le message de ce merge reste celui produit par git
