# Spec Delta

## ADDED Requirements

### Requirement: Colonne Verifying sous In Progress
Le Kanban SHALL afficher une colonne **Verifying** contenant les changes dont `kanban_status` vaut `verifying`. Cette colonne SHALL occuper le slot de **In Progress**, empilée sous elle, sans ajouter de slot horizontal : **In Progress** occupe le haut du slot en `flex-1 min-h-0`, prioritaire sur l'espace vertical, et **Verifying** le bas, avec une hauteur plafonnée à 40 % du slot et un défilement interne, séparées par un trait horizontal fin, comme le slot Done / Archived. La colonne SHALL être affichée lorsque la vérification est activée pour le workspace (valeur résolue sans réglage de change, voir `verification-settings`) ou lorsqu'elle contient au moins une carte ; elle SHALL être masquée sinon, **In Progress** reprenant alors tout le slot. Chaque carte SHALL afficher le nom du change, la progression de ses tâches et un badge d'état : `queued` (en attente), `running` (en cours, avec l'étape), `failed` (échec) ou `passed` (réussi, en attente de finalisation). Les cartes `failed` SHALL proposer des boutons d'action rapide « Relancer » et « Finaliser », qui n'ouvrent pas le détail et ne déclenchent pas de drag. Un clic sur une carte SHALL ouvrir le DetailPanel.

#### Scenario: Colonne empilée sous In Progress
- **WHEN** le Kanban est affiché avec au moins un change `verifying`
- **THEN** le slot de In Progress contient In Progress en haut, un trait horizontal, puis Verifying en bas, et le Kanban garde six slots horizontaux

#### Scenario: Hauteur plafonnée
- **WHEN** la colonne Verifying contient beaucoup de cartes
- **THEN** elle occupe au plus 40 % du slot et défile en interne, In Progress gardant l'espace résiduel

#### Scenario: Colonne masquée
- **WHEN** la vérification n'est activée pour le workspace et qu'aucun change n'est `verifying`
- **THEN** la colonne Verifying n'est pas affichée et In Progress occupe tout le slot

#### Scenario: Colonne affichée par son contenu
- **WHEN** la vérification est désactivée pour le workspace mais qu'un change porte encore le marqueur de vérification
- **THEN** la colonne Verifying est affichée avec ce change

#### Scenario: Badge d'état
- **WHEN** une vérification de conformité tourne pour un change
- **THEN** sa carte dans Verifying affiche le badge « en cours » avec l'étape `conformity`

#### Scenario: Actions d'une carte en échec
- **WHEN** un change `failed` est affiché
- **THEN** sa carte propose « Relancer » et « Finaliser », et un clic sur l'un d'eux n'ouvre pas le DetailPanel

#### Scenario: Carte non déplaçable
- **WHEN** l'utilisateur tente de saisir une carte de la colonne Verifying
- **THEN** le drag ne démarre pas

#### Scenario: Retour dans In Progress
- **WHEN** la vérification d'un change réussit et que son worker de finalisation démarre
- **THEN** la carte quitte la colonne Verifying pour In Progress, avec le badge du worker
