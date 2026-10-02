# Spec Delta

## MODIFIED Requirements

### Requirement: Approuver et fusionner un change en revue
Le backend SHALL exposer `POST /api/workspaces/{id}/changes/{name}/review/approve`. Pour un change dont le statut est `to-review` et sans worker actif, il SHALL fusionner `feature/<change>` dans la branche actuellement extraite du dépôt (fusion `--no-ff`), supprimer ensuite le worktree puis la branche, et répondre `200` avec la branche cible. Le marqueur de revue disparaissant avec la branche, le change SHALL passer en `done` (le `tasks.md` fusionné est coché) et un événement `change_updated` SHALL être publié. L'approbation SHALL NOT supposer que le pool d'agents tourne et SHALL NOT lancer de session d'agent. Elle SHALL refuser avec `409` un change qui n'est pas en revue (code `not_in_review`) ou sur lequel un worker est actif (`worker_active`). En cas de refus ou d'échec survenant avant la fusion, la branche, le worktree, le travail committé et le marqueur de revue SHALL être conservés tels qu'avant l'appel, et la réponse SHALL porter un code d'erreur et un message lisibles. Si la fusion a eu lieu mais que la suppression du worktree ou de la branche échoue, le backend SHALL NOT présenter cette situation comme un échec : il SHALL lever le marqueur de revue (le change passe en `done`), publier `change_updated` et répondre `200` avec la branche cible et un avertissement `cleanup_incomplete` portant un message lisible et la liste `remaining` de ce qui reste à nettoyer à la main (parmi `worktree`, `branch` et `marker`). Si le marqueur ne peut lui-même pas être levé, `remaining` SHALL contenir `marker`, et une nouvelle approbation de ce change SHALL reconnaître que sa branche est déjà fusionnée dans la branche cible, sans nouvelle intégration, validation ni fusion, puis terminer le nettoyage.

#### Scenario: Approbation réussie
- **WHEN** l'utilisateur approuve `add-user-auth`, en revue, alors que `main` n'a pas avancé
- **THEN** `feature/add-user-auth` est fusionnée dans `main`, le worktree puis la branche sont supprimés, la réponse est `200` avec la branche `main`, et le change passe en `done`

#### Scenario: Approbation avec le pool arrêté
- **WHEN** l'utilisateur approuve un change en revue alors que le pool du workspace est arrêté
- **THEN** l'approbation aboutit comme si le pool tournait

#### Scenario: Change qui n'est pas en revue
- **WHEN** l'utilisateur appelle l'approbation d'un change au statut `todo`
- **THEN** la réponse est `409` avec le code `not_in_review` et rien n'est modifié

#### Scenario: Worker actif sur le change
- **WHEN** l'approbation est demandée alors qu'un worker est actif sur ce change
- **THEN** la réponse est `409` avec le code `worker_active` et rien n'est modifié

#### Scenario: Approbations concurrentes
- **WHEN** deux approbations du même change arrivent simultanément
- **THEN** une seule fusionne ; l'autre reçoit `409` avec le code `not_in_review` ou un refus de fusion en cours, sans second merge

#### Scenario: Fusion réussie mais worktree non supprimable
- **WHEN** l'utilisateur approuve `add-user-auth` alors que son worktree contient un fichier non suivi qui fait refuser la suppression du worktree, et que la fusion réussit
- **THEN** la réponse est `200` avec la branche cible et un avertissement `cleanup_incomplete` dont `remaining` contient `worktree`, le marqueur de revue est levé et le change passe en `done`

#### Scenario: Fusion réussie mais branche non supprimable
- **WHEN** la fusion réussit mais que la suppression de la branche `feature/add-user-auth` échoue
- **THEN** la réponse est `200` avec un avertissement `cleanup_incomplete` dont `remaining` contient `branch`, le marqueur est levé et le change passe en `done`

#### Scenario: Marqueur non levable après la fusion
- **WHEN** la fusion réussit puis que la levée du marqueur échoue
- **THEN** la réponse est `200` avec un avertissement `cleanup_incomplete` dont `remaining` contient `marker`, et une nouvelle approbation du même change reconnaît la branche comme déjà fusionnée, sans rejouer la validation ni fusionner, puis termine le nettoyage et lève le marqueur

### Requirement: Confirmation et suivi de l'approbation dans l'interface
Le bouton « Approuver & Fusionner » de l'onglet Actions et le drag d'une carte de To Review vers Done SHALL demander une confirmation nommant le change et la branche cible avant d'appeler l'approbation. Pendant l'appel, le bouton SHALL être désactivé avec un indicateur de chargement et les actions de revue du change SHALL NOT pouvoir être déclenchées à nouveau. En cas de succès, la carte SHALL passer en Done et le DetailPanel SHALL se mettre à jour. Si la réponse de succès porte un avertissement `cleanup_incomplete`, le dialogue SHALL se fermer comme pour un succès, aucune erreur SHALL être affichée, et un message d'avertissement, dans la langue de l'interface, SHALL indiquer ce qui reste à nettoyer à la main d'après `remaining`. En cas d'échec, la carte SHALL rester (ou revenir) dans To Review et un message lisible, selon le code d'erreur renvoyé (conflit, validation en échec avec sa sortie, branche de base, merge en cours), SHALL être affiché sans fermer le DetailPanel.

#### Scenario: Confirmation avant fusion
- **WHEN** l'utilisateur clique sur « Approuver & Fusionner » ou dépose une carte To Review sur Done
- **THEN** une confirmation nommant le change et la branche cible s'affiche avant toute fusion

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur annule la confirmation
- **THEN** rien n'est fusionné et la carte reste dans To Review

#### Scenario: Fusion en cours
- **WHEN** l'approbation est en cours
- **THEN** le bouton est désactivé avec un indicateur de chargement

#### Scenario: Approbation réussie
- **WHEN** le backend répond `200`
- **THEN** la carte apparaît dans la colonne Done

#### Scenario: Approbation en échec
- **WHEN** le backend répond `409` avec le code `integration_conflict` ou `422` avec le code `validation_failed`
- **THEN** la carte reste dans To Review et l'onglet Actions affiche un message correspondant au code, avec la sortie de validation le cas échéant

#### Scenario: Approbation réussie avec nettoyage incomplet
- **WHEN** le backend répond `200` avec un avertissement `cleanup_incomplete`
- **THEN** le dialogue se ferme, la carte apparaît dans la colonne Done, aucune erreur n'est affichée et un message d'avertissement, dans la langue de l'interface, précise ce qui reste à nettoyer d'après `remaining`
