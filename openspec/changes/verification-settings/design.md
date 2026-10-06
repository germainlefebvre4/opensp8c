# Design

## Context

Voir `proposal.md` (Why). Ce change ne lance aucune vérification : il fournit la valeur résolue que `verification-stage` et `ui-verify-step` liront. État actuel observé :

- Les réglages à héritage suivent un moule unique dans `internal/preferences` : un défaut de Configuration (`PoolSettings`, zéro = valeur intégrée), une surcharge par workspace à pointeurs (`PoolOverride`, `nil` = hériter) dans `WorkspacePrefs`, une résolution à l'appel (`ResolvePool`), et des patchs partiels à drapeaux `Set` / `Reset` (`applyPoolPatch`). `GET/PATCH /workspaces/{id}/settings` renvoie `{ overrides, inherited, resolved }` ; `PATCH /api/preferences` porte `agentSettings` et `poolDefaults`.
- Il n'existe pas de niveau « change » pour les réglages. Le seul stockage par change est `.openspec.yaml` (champs `launched`, `order`, `tags`, `dependencies`), lu par `loadChange` et réécrit par `SetLaunched`, `ClearKanbanState`, `ReorderReady` et le tagger via la struct `openspecMeta` : un champ absent de la struct serait perdu à la première réécriture.
- `watcher.go` surveille déjà `.openspec.yaml` : une écriture produit `change_updated` sans code supplémentaire.
- Les rôles d'agent sont une liste fermée (`preferences.Roles`), avec préréglages par agent (`Preset`) et cascade de résolution (`levels`).
- `ConfigurationPage` et `SettingsPage` partagent leurs formulaires par une prop `scope` (`global` / `workspace`), comme `AgentPoolSettingsForm` et `RoleSettingsTable`.

## Goals / Non-Goals

**Goals:**
- Une résolution unique, pure et testable, partagée par les deux futures étapes, sans dépendance au pool ni au worktree.
- Réutiliser les mécanismes d'héritage existants plutôt qu'en introduire un nouveau.
- Rendre le réglage de change visible et modifiable là où l'utilisateur regarde le change.

**Non-Goals:**
- Exécuter ou planifier une vérification, marqueur persistant, colonne `Verifying`, politique d'échec : changes suivants.
- Réglage du change pour `uiStartCommand` / `uiBaseUrl` (décrivent le projet, pas un change).
- Un mode tri-état au niveau Configuration : `off` et absence y sont équivalents.

## Decisions

### D1. Paquet `internal/verification` pour les types et la résolution

Un paquet minuscule porte les types partagés et la fonction de résolution :

```go
// Override : valeurs d'un niveau ; nil = hériter.
type Override struct {
    Conformity *bool
    UI         *bool
}
type LaunchParams struct { UIStartCommand, UIBaseURL string }
type Resolved struct { Conformity, UI bool; LaunchParams }

func Resolve(platform, workspace, change *Override, ...) Resolved
```

`openspec` (champ `verification` de `openspecMeta` et du détail) et `preferences` (défauts et surcharge workspace) l'importent tous deux ; ni l'un ni l'autre ne dépend de l'autre. `Resolve` est une fonction pure : cascade `change > workspace > plateforme > false`, étape par étape.

*Alternative :* mettre la résolution dans `preferences`. Écartée : `openspec` devrait importer `preferences` (ou l'inverse) juste pour un type, et le niveau « change » ne ressemble pas aux préférences (fichier versionné du dépôt, pas `preferences.json`).

### D2. Configuration et workspace : `bool` en pointeurs, comme le pool

`Preferences.VerificationDefaults *VerificationSettings` (champs `Conformity`, `UI` booléens simples, plus `UIStartCommand`, `UIBaseURL`, tous `omitempty` : `false` = absent, ce qui exprime « off = pas de réglage » du niveau Configuration). `WorkspacePrefs.Verification *VerificationOverride` à pointeurs (`*bool`, `*string`), pour distinguer « désactivé explicitement » de « hérité ». `isEmpty` et `clone` de `WorkspacePrefs` sont étendus ; une section vide est retirée du fichier, comme pour le pool. Le patch suit `PoolPatch` (drapeaux `Set` / `Reset`, `null` = réinitialiser) avec une fonction `applyVerificationPatch` calquée sur `applyPoolPatch`.

`GET /workspaces/{id}/settings` gagne `verification` dans `overrides`, `inherited` et `resolved`, avec la même structure que le pool. `PATCH /api/preferences` gagne `verificationDefaults` et passe par `ValidateGlobalUpdate` avant la première écriture (validation de tout avant d'écrire, comme aujourd'hui).

### D3. Réglage de change dans `.openspec.yaml`, lu dans le dépôt principal

```yaml
verification:
  conformity: true
  ui: false
```

`openspecMeta` gagne `Verification *verification.Override` (`yaml:"verification,omitempty"`). Comme `SetLaunched`, `ClearKanbanState`, `ReorderReady` et le tagger désérialisent puis resérialisent la struct, le champ est préservé par construction dès qu'il y figure ; un test de non-régression par fonction d'écriture le garantit. Le champ est retiré (pointeur remis à `nil`) quand les deux valeurs sont retirées, pour ne pas laisser un `verification: {}` résiduel.

La lecture se fait toujours depuis `<workspacePath>/openspec/changes/<name>/.openspec.yaml`, jamais depuis le worktree, qui est figé au dernier commit et ne verrait pas une modification postérieure au lancement (voir la spec, « Réglage du change lu dans le dépôt principal »). Pas de verrou propre : la fenêtre read-modify-write d'un `.openspec.yaml` est du même ordre que celle des écritures existantes.

*Alternatives :* (a) marqueur dans la configuration git comme la revue : non versionné, ne suit pas le change, et inutilement opaque pour un réglage éditable ; (b) un fichier séparé par change : plus de fichiers, aucun gain, `.openspec.yaml` est déjà le lieu des métadonnées de change.

### D4. Endpoint dédié pour le change, valeurs `true` / `false` / `null`

`PATCH /workspaces/{id}/changes/{name}/verification` accepte `{ "conformity": true|false|null, "ui": ... }`. Le décodage distingue champ absent (inchangé) de `null` (retrait) ; un type à drapeaux `Set` / `Reset` identique à `PoolPatch` est réutilisé. Tout champ inconnu (notamment `uiStartCommand`, `uiBaseUrl`) est rejeté en `400` (`DisallowUnknownFields`). Le handler résout le workspace, vérifie l'existence du change actif (`404`), refuse l'archivé (`409`), écrit via une fonction `openspec.SetVerification(changeRoot, patch)`, et répond avec le même objet `verification` que le détail, recalculé après écriture.

Le détail d'un change (`ChangeDetail`) gagne `Verification` (`override`, `inherited`, `resolved`) : `GetChangeDetail` ne connaissant pas les préférences, la résolution est faite par le handler, qui dispose du service de préférences, comme pour `worker_blocked_reason`.

*Alternative :* faire passer le réglage de change par `PATCH …/launch` : mélange deux responsabilités sans rapport et réutiliserait un endpoint à 204.

### D5. Sixième rôle `verifier`

`Role` gagne `RoleVerifier` dans `Roles`, après `fixer` (ordre d'affichage) ; `Valid()` l'accepte, `Preset` lui donne `sonnet` / `medium` pour Claude. Le rôle est ajouté à `levels` sans autre changement : la cascade est générique. Aucune clé existante n'est touchée, donc aucun `preferences.json` n'a à être migré. Les écrans parcourent déjà `Roles` ; seul le libellé de l'étape correspondante est à ajouter (« Vérification »).

Un seul rôle couvre `conformity` et `ui`. Distinguer un modèle économique pour l'une et un modèle plus fort pour l'autre est possible plus tard en scindant le rôle ; ce n'est pas nécessaire pour poser la configuration.

### D6. Écrans : un formulaire, deux portées

Un composant `VerificationSettingsForm` reçoit `scope` (`global` / `workspace`), les valeurs stockées, les valeurs héritées et un `onSave`, comme `AgentPoolSettingsForm`. En portée `global`, deux interrupteurs on/off ; en portée `workspace`, un sélecteur à trois choix (Hérité / Activé / Désactivé) par étape, avec la valeur héritée affichée en regard de « Hérité ». Les champs `uiStartCommand` et `uiBaseUrl` se comportent comme `validationCommand` (valeur héritée en indication, surcharge distinguée, réinitialisation). La validation côté client (URL absolue `http` / `https`) réutilise la même règle que le backend.

Le sélecteur à trois choix est un composant partagé (`TriStateSelect`) réutilisé par la section Vérification du DetailPanel, qui l'alimente avec `verification.override` et `verification.inherited` du détail et appelle un hook de mutation `useSetChangeVerification` (invalidation de `['change-detail', id, name]` au succès ; le watcher invalide le reste via `change_updated`).

## Risks / Trade-offs

- [Un champ absent de `openspecMeta` serait effacé à la première réécriture du `.openspec.yaml`] → le champ est ajouté à la struct, avec un test de préservation pour chaque fonction d'écriture (`SetLaunched`, `ClearKanbanState`, `ReorderReady`, tagger).
- [Le réglage de change est lu dans le dépôt principal alors que le worktree contient une ancienne copie du `.openspec.yaml`] → un seul point de lecture, `verification.Resolve` appelé avec la valeur du dépôt principal ; aucune lecture du worktree dans ce code.
- [Réglage activé sans commande de lancement UI] → l'étape `ui` pourra être activée sans `uiStartCommand` ; ce change ne l'empêche pas (c'est à `ui-verify-step` de signaler l'absence de paramètre avec un message clair), mais l'écran l'indique par un avertissement discret lorsque `ui` est résolue à `on` sans commande.
- [Six rôles dans les écrans et les specs existantes qui disent « cinq »] → les deltas modifient les exigences concernées ; le Purpose d'`agent-role-settings` (« exploration, fast-forward, … ») ne peut pas être modifié par delta : il est mis à jour à l'archivage.
- [Aucun effet visible tant que les étapes ne sont pas implémentées] → assumé : le change est livrable et testable seul (valeur résolue exposée par l'API et affichée dans les écrans), et évite que les deux changes suivants dupliquent l'héritage.

## Migration Plan

Aucune migration : tous les champs sont optionnels, l'absence équivaut à `off` / hériter, et un `preferences.json` ou un `.openspec.yaml` existant se charge sans erreur. Retour arrière : retirer les champs ; les fichiers qui les contiennent restent lisibles (les clés YAML / JSON inconnues sont ignorées par les versions antérieures du chargement de `preferences.json` ; pour `.openspec.yaml`, une version antérieure les perdrait à la prochaine réécriture, ce qui revient à « hériter »).
