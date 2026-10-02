# change-review-panel Specification

## Purpose

Permet de relire, dans l'application, le travail d'un change en revue (fichiers modifiés et diff de sa branche) avant de l'approuver ou de demander une correction, sans ouvrir le worktree à la main.

## Requirements

### Requirement: Liste des fichiers modifiés d'un change en revue
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/review` qui, pour un change au statut `to-review`, retourne la branche du change (`feature/<change>`), la branche de base, un indicateur `target_ahead` (la branche cible contient des commits absents de la branche du change) et la liste des fichiers modifiés par la branche par rapport à son point de divergence avec la branche de base. Chaque fichier SHALL porter son chemin, son statut (`added`, `modified` ou `deleted`), son nombre de lignes ajoutées et supprimées et un indicateur `binary`. La branche de base est celle enregistrée pour le change, à défaut la branche actuellement extraite du dépôt. Un renommage SHALL apparaître comme une suppression et un ajout. L'endpoint SHALL répondre `404` pour un change inconnu et `409` avec le code `not_in_review` pour un change qui n'est pas en revue. Il SHALL être en lecture seule, calculé à partir de la branche (et non du worktree), et fonctionner lorsque le pool est arrêté ou que le worktree a disparu.

#### Scenario: Change en revue avec des fichiers modifiés
- **WHEN** le client appelle `GET .../changes/add-user-auth/review` pour un change en revue dont la branche modifie 2 fichiers et en ajoute 1
- **THEN** la réponse contient la branche `feature/add-user-auth`, la branche de base, `target_ahead`, et 3 fichiers avec chemin, statut et compteurs de lignes

#### Scenario: Fichier binaire
- **WHEN** la branche ajoute une image
- **THEN** le fichier est listé avec `binary` à `true` et des compteurs de lignes à 0

#### Scenario: Renommage
- **WHEN** la branche renomme un fichier
- **THEN** la liste contient une suppression du chemin d'origine et un ajout du nouveau chemin

#### Scenario: Branche cible avancée
- **WHEN** `main` contient un commit absent de `feature/add-user-auth`
- **THEN** `target_ahead` vaut `true`, et les fichiers listés sont uniquement ceux modifiés par la branche (pas ceux de ce commit)

#### Scenario: Change qui n'est pas en revue
- **WHEN** le client appelle l'endpoint pour un change au statut `todo`
- **THEN** la réponse est `409` avec le code `not_in_review`

#### Scenario: Pool arrêté et worktree absent
- **WHEN** le pool est arrêté et le worktree du change supprimé, la branche existant toujours
- **THEN** la liste des fichiers est quand même retournée

### Requirement: Patch d'un fichier d'un change en revue
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/review/diff?path=<chemin>` qui retourne le patch unifié du fichier indiqué entre le point de divergence avec la branche de base et `feature/<change>`. Le paramètre `path` SHALL désigner un fichier présent dans la liste de la revue : un chemin absent, absolu ou contenant `..` SHALL être refusé (`400` pour un chemin invalide, `404` pour un fichier hors de la revue). Pour un fichier binaire, la réponse SHALL indiquer `binary` sans patch. Un patch dépassant 512 Kio SHALL être tronqué avec l'indicateur `truncated` à `true`. L'endpoint SHALL répondre `409` avec le code `not_in_review` pour un change qui n'est pas en revue, et ne modifier ni la branche ni le worktree.

#### Scenario: Patch d'un fichier modifié
- **WHEN** le client demande le patch d'un fichier modifié par la branche
- **THEN** la réponse contient un patch unifié avec les hunks et les lignes ajoutées et supprimées de ce fichier

#### Scenario: Chemin hors de la revue
- **WHEN** le client demande le patch d'un fichier que la branche ne modifie pas
- **THEN** la réponse est `404` et aucun contenu du dépôt n'est renvoyé

#### Scenario: Tentative de sortie du dépôt
- **WHEN** le client passe `path=../../etc/passwd` ou un chemin absolu
- **THEN** la réponse est `400`

#### Scenario: Fichier binaire
- **WHEN** le client demande le patch d'une image ajoutée
- **THEN** la réponse indique `binary` à `true` sans patch

#### Scenario: Patch volumineux
- **WHEN** le patch d'un fichier dépasse 512 Kio
- **THEN** la réponse contient un patch tronqué et `truncated` à `true`

### Requirement: Contenu final d'un fichier d'un change en revue
Le backend SHALL exposer `GET /api/workspaces/{id}/changes/{name}/review/file?path=<chemin>` qui retourne le contenu du fichier tel qu'il est dans `feature/<change>`, avec les mêmes règles de validation de `path`, de statut `409`/`404`/`400`, de lecture seule et de limite de taille (contenu tronqué avec `truncated` au-delà de 512 Kio, `binary` sans contenu pour un fichier binaire) que le patch. Un fichier supprimé par la branche SHALL répondre `404`.

#### Scenario: Contenu d'un fichier Markdown
- **WHEN** le client demande le contenu d'un `proposal.md` modifié par la branche
- **THEN** la réponse contient le texte du fichier à sa version dans `feature/<change>`

#### Scenario: Fichier supprimé par la branche
- **WHEN** le client demande le contenu d'un fichier supprimé par la branche
- **THEN** la réponse est `404`

### Requirement: Onglet Revue dans le DetailPanel
Le `DetailPanel` SHALL afficher un onglet **Revue** pour un change au statut `to-review`, et uniquement pour lui. L'onglet SHALL lister les fichiers modifiés en deux groupes, **OpenSpec** (tout chemin sous `openspec/`) puis **Code** (les autres), chaque ligne affichant le chemin, le statut et les compteurs `+`/`-` ; un groupe vide SHALL être masqué. Il SHALL afficher la branche du change et la branche de base, et un avertissement lorsque `target_ahead` est vrai (la branche cible a avancé : l'approbation intégrera ces commits). Un clic sur un fichier SHALL déplier son diff, chargé à la demande, avec les lignes ajoutées et supprimées distinguées visuellement et les numéros de lignes des hunks ; un fichier binaire SHALL afficher un message au lieu d'un diff, un patch tronqué SHALL le signaler. L'onglet SHALL gérer les états de chargement, d'erreur (message lisible, nouvelle tentative possible) et de revue sans fichier modifié (message dédié). Si le change quitte le statut `to-review` alors que l'onglet est ouvert, l'onglet SHALL disparaître et le panneau SHALL revenir à l'onglet Tâches. Les libellés SHALL être disponibles en français et en anglais.

#### Scenario: Onglet présent uniquement en revue
- **WHEN** le DetailPanel s'ouvre pour un change au statut `to-review`, puis pour un change au statut `todo`
- **THEN** l'onglet Revue est visible pour le premier et absent pour le second

#### Scenario: Fichiers groupés
- **WHEN** la revue contient `openspec/changes/add-user-auth/tasks.md` et `backend/auth.go`
- **THEN** le groupe OpenSpec contient le premier et le groupe Code le second

#### Scenario: Dépliage d'un diff
- **WHEN** l'utilisateur clique sur un fichier de la liste
- **THEN** son diff est chargé puis affiché, lignes ajoutées et supprimées distinguées, et un second clic le replie

#### Scenario: Fichier binaire
- **WHEN** l'utilisateur déplie un fichier binaire
- **THEN** un message indique qu'aucun diff n'est affichable pour ce fichier

#### Scenario: Branche cible avancée
- **WHEN** `target_ahead` est vrai
- **THEN** l'onglet affiche un avertissement indiquant que la branche cible a avancé

#### Scenario: Revue sans fichier
- **WHEN** la branche ne modifie aucun fichier par rapport à la base
- **THEN** l'onglet affiche un message de revue vide

#### Scenario: Erreur de chargement
- **WHEN** la requête de la liste des fichiers échoue
- **THEN** l'onglet affiche un message d'erreur et un moyen de réessayer

#### Scenario: Change qui quitte la revue
- **WHEN** l'onglet Revue est actif et que le change est approuvé ou renvoyé en correction
- **THEN** l'onglet disparaît et le panneau affiche l'onglet Tâches

### Requirement: Rendu des fichiers Markdown dans la revue
Pour un fichier `.md` non supprimé, l'onglet Revue SHALL proposer une bascule entre le **diff** et le **rendu** du fichier à sa version finale, en réutilisant le rendu Markdown existant de la plateforme ; le diff reste la vue par défaut. Le rendu SHALL charger le contenu à la demande et afficher un message en cas d'échec ou de contenu tronqué.

#### Scenario: Bascule vers le rendu
- **WHEN** l'utilisateur déplie `proposal.md` et active le rendu
- **THEN** le contenu final du fichier est affiché rendu en Markdown

#### Scenario: Retour au diff
- **WHEN** l'utilisateur repasse de la vue rendue au diff
- **THEN** le diff du fichier est de nouveau affiché

#### Scenario: Fichier non Markdown
- **WHEN** l'utilisateur déplie un fichier de code
- **THEN** aucune bascule de rendu n'est proposée

### Requirement: Lien vers le dernier run du worker depuis la revue
L'onglet Revue SHALL afficher, pour le change, son dernier run du pool d'agents (date et issue) et un lien ouvrant le détail de ce run dans la vue Agents. S'il n'existe aucun run pour ce change, aucun lien SHALL être affiché.

#### Scenario: Run disponible
- **WHEN** le change en revue possède un run du pool avec l'issue `awaiting-review`
- **THEN** l'onglet affiche la date et l'issue de ce run et un lien qui ouvre son détail dans la vue Agents

#### Scenario: Aucun run
- **WHEN** aucun run n'est enregistré pour le change
- **THEN** aucun lien de run n'est affiché
