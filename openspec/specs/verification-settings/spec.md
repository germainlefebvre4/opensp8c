# verification-settings Specification

## Purpose
Permet d'activer ou non, à la plateforme, au workspace ou à un change, les étapes de vérification automatique d'un change (conformité et UI), qui consomment beaucoup de tokens, avec héritage entre niveaux et désactivation par défaut.

## Requirements

### Requirement: Deux étapes de vérification configurables
Le système SHALL distinguer deux étapes de vérification automatique d'un change, configurables indépendamment l'une de l'autre : `conformity` (vérification de la conformité du code, des specs et des tâches) et `ui` (vérification par un parcours dans l'application lancée). Chaque étape SHALL pouvoir être activée ou désactivée sans que l'état de l'autre change.

#### Scenario: Étapes indépendantes
- **WHEN** l'utilisateur active `conformity` pour un workspace sans toucher à `ui`
- **THEN** `conformity` est activée pour ce workspace et `ui` garde sa valeur précédente

### Requirement: Trois niveaux de réglage avec héritage
Pour chaque étape, le système SHALL accepter, à chacun des trois niveaux (Configuration de la plateforme, workspace, change), l'une des valeurs `on`, `off` ou « hériter » (absence de valeur). La valeur intégrée, quand aucun niveau ne définit de valeur, SHALL être `off` pour les deux étapes. Le niveau de Configuration ne SHALL proposer que `on` et `off`, `off` valant absence de réglage.

#### Scenario: Aucune valeur définie nulle part
- **WHEN** ni la Configuration, ni le workspace, ni le change ne définissent de valeur pour `ui`
- **THEN** l'étape `ui` est résolue à `off`

#### Scenario: Valeur de Configuration héritée
- **WHEN** la Configuration active `conformity` et que ni le workspace ni le change ne définissent de valeur
- **THEN** `conformity` est résolue à `on` pour ce change

### Requirement: Résolution en cascade change, workspace, plateforme
Pour chaque étape, le système SHALL résoudre la valeur effective en retenant la première valeur définie parmi, dans cet ordre : le réglage du change, la surcharge du workspace, le défaut de la Configuration, la valeur intégrée `off`. Chaque étape SHALL être résolue séparément. La valeur du change SHALL être lue dans le dépôt principal du workspace, et non dans le worktree d'un worker. La valeur SHALL être résolue au moment où l'étape est évaluée : une modification ultérieure n'affecte pas une étape déjà démarrée, mais s'applique aux évaluations suivantes.

#### Scenario: Le change surcharge le workspace
- **WHEN** le workspace active `ui` et que le change définit `ui` à `off`
- **THEN** `ui` est résolue à `off` pour ce change
- **THEN** un autre change du même workspace sans réglage résout `ui` à `on`

#### Scenario: Le change active malgré un défaut désactivé
- **WHEN** la Configuration et le workspace ne définissent rien et que le change définit `conformity` à `on`
- **THEN** `conformity` est résolue à `on` pour ce change uniquement

#### Scenario: La surcharge du workspace prime sur la Configuration
- **WHEN** la Configuration active `conformity` et que le workspace `A` la désactive
- **THEN** `conformity` est résolue à `off` pour les changes de `A`
- **THEN** elle est résolue à `on` pour les changes d'un workspace `B` sans surcharge

#### Scenario: Étapes résolues séparément
- **WHEN** le change définit `conformity` à `on` et laisse `ui` hériter d'une Configuration qui ne définit rien
- **THEN** `conformity` est résolue à `on` et `ui` à `off`

#### Scenario: Réglage du change lu dans le dépôt principal
- **WHEN** l'utilisateur modifie le réglage d'un change dont un worker occupe le worktree
- **THEN** la résolution suivante utilise la valeur du dépôt principal, que le worktree ne contienne pas encore cette modification ou en contienne une ancienne copie

#### Scenario: Modification pendant une étape en cours
- **WHEN** l'utilisateur désactive `ui` pour un change dont l'étape `ui` est déjà démarrée
- **THEN** l'étape en cours n'est pas interrompue par ce réglage
- **THEN** les évaluations suivantes résolvent `ui` à `off`

### Requirement: Paramètres de lancement de la vérification UI
Le système SHALL permettre de définir, à la Configuration et au workspace seulement, une commande de lancement de l'application (`uiStartCommand`, texte libre) et une URL de base (`uiBaseUrl`) utilisées par l'étape `ui`. Le workspace SHALL surcharger la Configuration champ par champ. Un champ vide après suppression des espaces SHALL équivaloir à une valeur absente. `uiBaseUrl`, lorsqu'elle est définie, SHALL être une URL absolue `http` ou `https`, dont le port peut être remplacé par le jeton `{port}` (par exemple `http://localhost:{port}`). `uiStartCommand` et `uiBaseUrl` peuvent contenir le jeton `{port}`, que l'étape `ui` remplace par un port libre choisi au lancement. Ces paramètres ne SHALL pas être définissables au niveau d'un change.

#### Scenario: Le workspace surcharge un seul paramètre
- **WHEN** la Configuration définit `uiStartCommand` à `make dev` et `uiBaseUrl` à `http://localhost:5173`, et que le workspace `A` ne surcharge que `uiBaseUrl` avec `http://localhost:3000`
- **THEN** la résolution pour `A` donne `make dev` et `http://localhost:3000`

#### Scenario: URL de base invalide
- **WHEN** l'utilisateur tente d'enregistrer `uiBaseUrl` à `localhost:5173` ou à `ftp://hote`
- **THEN** l'enregistrement est refusé avec une erreur de validation et les réglages existants ne sont pas modifiés

#### Scenario: Jeton de port accepté
- **WHEN** l'utilisateur enregistre `uiBaseUrl` à `http://localhost:{port}` et `uiStartCommand` à `npm run dev -- --port {port}`
- **THEN** l'enregistrement réussit et les valeurs sont retournées telles quelles, sans substitution

#### Scenario: Jeton mal placé
- **WHEN** l'utilisateur enregistre `uiBaseUrl` à `{port}://localhost`
- **THEN** l'enregistrement est refusé avec une erreur de validation

#### Scenario: Valeur vide
- **WHEN** l'utilisateur enregistre `uiStartCommand` avec une chaîne composée d'espaces
- **THEN** le paramètre est considéré absent à ce niveau

#### Scenario: Pas de paramètre de lancement au niveau du change
- **WHEN** une requête de réglage de change contient `uiStartCommand` ou `uiBaseUrl`
- **THEN** la requête est rejetée avec une erreur de validation

### Requirement: Persistance rétrocompatible des réglages de vérification
Les défauts de Configuration SHALL être persistés dans `preferences.json` et retournés par `GET /api/preferences`. La surcharge d'un workspace SHALL être persistée dans la section de ce workspace de `preferences.json`. Le réglage d'un change SHALL être persisté dans son `.openspec.yaml`. L'absence de ces champs SHALL équivaloir à « hériter » (ou `off` pour la Configuration), sans migration ni erreur au chargement d'un fichier existant. Une section ou un champ devenu vide SHALL être retiré du fichier. L'écriture du `.openspec.yaml` par une autre action (lancement, retrait du lancement, réinitialisation de l'état du Kanban, retagging) SHALL préserver le réglage de vérification du change.

#### Scenario: Fichiers existants sans réglage de vérification
- **WHEN** l'application démarre avec un `preferences.json` et des `.openspec.yaml` sans champ de vérification
- **THEN** le chargement réussit et les deux étapes sont résolues à `off`

#### Scenario: Réglage du change préservé par un lancement
- **WHEN** le change a `conformity` à `on` dans son `.openspec.yaml` et que l'utilisateur le lance puis le retire du lancement
- **THEN** son `.openspec.yaml` porte toujours `conformity` à `on`

#### Scenario: Réinitialisation de l'état du Kanban
- **WHEN** l'état « lancé » et l'ordre d'un change sont réinitialisés
- **THEN** le réglage de vérification du change est conservé

#### Scenario: Retrait d'une surcharge de workspace
- **WHEN** l'utilisateur réinitialise la dernière surcharge de vérification d'un workspace qui n'a aucune autre surcharge
- **THEN** la section de ce workspace est retirée de `preferences.json`

### Requirement: Validation et exposition des réglages de plateforme et de workspace
Le système SHALL accepter une mise à jour partielle des défauts de Configuration (`verificationDefaults`) et des surcharges d'un workspace (`verification`) via les points d'entrée de préférences et de réglages de workspace existants. Une valeur `null` SHALL réinitialiser un champ à l'héritage. Une valeur autre qu'un booléen pour une étape, ou un champ inconnu, SHALL être rejetée avec une erreur de validation sans modifier les réglages enregistrés. La réponse de `GET /workspaces/{id}/settings` SHALL inclure, pour la vérification, `overrides` (valeurs propres au workspace), `inherited` (valeurs de la Configuration, valeur intégrée comprise) et `resolved` (valeurs effectives pour ce workspace, sans réglage de change).

#### Scenario: Surcharge enregistrée
- **WHEN** l'utilisateur envoie `{"verification": {"conformity": true}}` pour le workspace `A`
- **THEN** la section de `A` dans `preferences.json` porte `conformity` à vrai
- **THEN** la réponse retourne cette valeur dans `overrides`, la valeur de Configuration dans `inherited` et `on` dans `resolved`

#### Scenario: Réinitialisation par null
- **WHEN** l'utilisateur envoie `{"verification": {"conformity": null}}` pour un workspace qui surchargeait `conformity`
- **THEN** la surcharge disparaît et `resolved` reprend la valeur de Configuration

#### Scenario: Valeur invalide
- **WHEN** l'utilisateur envoie `{"verification": {"ui": "maybe"}}`
- **THEN** la requête est rejetée avec une erreur de validation et aucun réglage n'est modifié

#### Scenario: Workspace inconnu
- **WHEN** l'utilisateur envoie une mise à jour pour un identifiant de workspace inconnu
- **THEN** la requête est rejetée avec le statut `404`

### Requirement: Réglage de vérification d'un change
Le système SHALL exposer `PATCH /workspaces/{id}/changes/{name}/verification` pour modifier le réglage de vérification d'un change actif. Le corps SHALL contenir `conformity` et/ou `ui` avec une valeur `true`, `false` ou `null` ; `null` SHALL retirer la valeur du change pour revenir à l'héritage, et un champ absent du corps SHALL rester inchangé. La réponse SHALL contenir la surcharge du change et les valeurs effectives résolues. Un change introuvable SHALL donner `404` ; un change archivé SHALL être refusé avec `409` sans modification.

#### Scenario: Activation pour un change
- **WHEN** l'utilisateur envoie `{"conformity": true}` pour un change actif
- **THEN** le `.openspec.yaml` du change porte `conformity` à vrai et la réponse indique `conformity` résolue à `on`
- **THEN** la valeur de `ui` du change reste inchangée

#### Scenario: Retour à l'héritage
- **WHEN** l'utilisateur envoie `{"ui": null}` pour un change qui définissait `ui`
- **THEN** le champ est retiré du `.openspec.yaml` et la réponse reflète la valeur héritée

#### Scenario: Dernier réglage retiré
- **WHEN** l'utilisateur retire les deux valeurs d'un change qui n'en avait pas d'autre
- **THEN** le `.openspec.yaml` ne contient plus de section de vérification

#### Scenario: Change archivé
- **WHEN** l'utilisateur envoie une mise à jour pour un change archivé
- **THEN** la requête est rejetée avec le statut `409` et le `.openspec.yaml` n'est pas modifié

#### Scenario: Change introuvable
- **WHEN** l'utilisateur envoie une mise à jour pour un nom de change inexistant
- **THEN** la requête est rejetée avec le statut `404`

### Requirement: Écran de réglage de la vérification dans Configuration
Configuration SHALL exposer un sous-onglet « Vérification » (« Verification » en anglais) permettant d'activer ou de désactiver `conformity` et `ui` par défaut pour tous les workspaces, et de définir `uiStartCommand` et `uiBaseUrl`. Chaque interrupteur SHALL être présenté comme désactivé par défaut, et le sous-onglet SHALL indiquer que ces étapes consomment des tokens. Les valeurs SHALL être enregistrées avec les mêmes règles de validation que l'API.

#### Scenario: Valeurs initiales
- **WHEN** l'utilisateur ouvre Configuration > Vérification sans réglage enregistré
- **THEN** les deux interrupteurs sont désactivés et les champs de lancement UI sont vides, avec une mention du coût en tokens

#### Scenario: Enregistrement
- **WHEN** l'utilisateur active `conformity`, saisit `make dev` comme commande de lancement et enregistre
- **THEN** `preferences.json` reflète ces valeurs et `GET /api/preferences` les retourne

#### Scenario: Valeur invalide
- **WHEN** l'utilisateur saisit une URL de base sans schéma et enregistre
- **THEN** une erreur de validation est affichée et les valeurs enregistrées ne sont pas modifiées

### Requirement: Écran de surcharge de la vérification dans Settings
Settings SHALL exposer un sous-onglet « Vérification » (« Verification » en anglais) permettant de surcharger, pour le workspace actif, chacune des deux étapes avec les choix « Hérité », « Activé » et « Désactivé », ainsi que `uiStartCommand` et `uiBaseUrl`. Pour une valeur non surchargée, l'écran SHALL afficher la valeur héritée de Configuration ; une surcharge SHALL être distinguée visuellement d'une valeur héritée et pouvoir être réinitialisée.

#### Scenario: Valeur héritée affichée
- **WHEN** Configuration active `conformity` et que le workspace n'a aucune surcharge
- **THEN** le choix « Hérité » est sélectionné et indique la valeur héritée « Activé »

#### Scenario: Surcharge marquée
- **WHEN** l'utilisateur choisit « Désactivé » pour `conformity` dans le workspace
- **THEN** le réglage est présenté comme une surcharge du workspace et `preferences.json` porte la valeur dans la section de ce workspace

#### Scenario: Réinitialisation
- **WHEN** l'utilisateur choisit « Hérité » pour une étape surchargée
- **THEN** la surcharge disparaît et l'écran affiche à nouveau la valeur héritée

### Requirement: Pilote de la vérification UI
Le système SHALL permettre de définir, à la Configuration et au workspace, le pilote de l'étape `ui` (`uiDriver`), c'est-à-dire la façon dont l'agent vérificateur dispose d'un navigateur. Les valeurs SHALL être `auto`, `playwright`, `chrome` et `custom`. `auto` SHALL être la valeur intégrée quand aucun niveau ne définit de valeur : la plateforme ne fournit alors aucun outil et l'agent utilise ce que la configuration de l'hôte lui offre. Le workspace SHALL surcharger la Configuration. Le pilote ne SHALL pas être définissable au niveau d'un change. Toute autre valeur SHALL être rejetée avec une erreur de validation sans modifier les réglages enregistrés.

#### Scenario: Valeur intégrée
- **WHEN** ni la Configuration ni le workspace ne définissent `uiDriver`
- **THEN** le pilote résolu est `auto`

#### Scenario: Le workspace surcharge la Configuration
- **WHEN** la Configuration définit `uiDriver` à `playwright` et que le workspace `A` le définit à `custom`
- **THEN** le pilote résolu pour `A` est `custom` et celui d'un workspace sans surcharge est `playwright`

#### Scenario: Valeur inconnue
- **WHEN** l'utilisateur envoie `{"verification": {"uiDriver": "selenium"}}`
- **THEN** la requête est rejetée avec une erreur de validation et aucun réglage n'est modifié

#### Scenario: Réinitialisation
- **WHEN** l'utilisateur envoie `{"verification": {"uiDriver": null}}` pour un workspace qui surchargeait le pilote
- **THEN** la surcharge disparaît et le pilote résolu reprend la valeur de Configuration

#### Scenario: Pas de pilote au niveau du change
- **WHEN** une requête de réglage de change contient `uiDriver`
- **THEN** la requête est rejetée avec une erreur de validation

### Requirement: Le pilote `chrome` est réservé au workspace
Le pilote `chrome` pilote le navigateur de l'utilisateur, avec ses sessions ouvertes. Il SHALL pouvoir être choisi au niveau d'un workspace seulement : une mise à jour des défauts de Configuration (`verificationDefaults`) contenant `uiDriver` à `chrome` SHALL être rejetée avec une erreur de validation sans modifier les réglages enregistrés. Une valeur `chrome` présente au niveau Configuration dans un `preferences.json` modifié à la main SHALL être ignorée à la résolution (le niveau est alors traité comme n'ayant pas de pilote), sans erreur au chargement.

#### Scenario: Refus à la Configuration
- **WHEN** l'utilisateur envoie `{"verificationDefaults": {"uiDriver": "chrome"}}`
- **THEN** la requête est rejetée avec une erreur de validation et les défauts enregistrés ne sont pas modifiés

#### Scenario: Choix par un workspace
- **WHEN** l'utilisateur envoie `{"verification": {"uiDriver": "chrome"}}` pour le workspace `A`
- **THEN** l'enregistrement réussit et le pilote résolu pour `A` est `chrome`, alors que celui des autres workspaces reste inchangé

#### Scenario: Fichier édité à la main
- **WHEN** `preferences.json` porte `uiDriver` à `chrome` dans `verificationDefaults` et que le workspace `B` n'a pas de surcharge
- **THEN** le chargement réussit et le pilote résolu pour `B` est `auto`

#### Scenario: Un workspace corrige une valeur invalide
- **WHEN** le workspace `C` surcharge `uiDriver` à `playwright` alors que la Configuration (éditée à la main) porte `chrome`
- **THEN** le pilote résolu pour `C` est `playwright`

### Requirement: Paramètres du pilote `custom`
Le système SHALL permettre de définir, à la Configuration et au workspace, `uiMcpConfig` (chemin d'un fichier de configuration MCP, texte libre) et `uiAllowedTools` (liste d'outils autorisés, chacun une chaîne non vide, 50 au plus). Le workspace SHALL surcharger la Configuration champ par champ ; `uiAllowedTools` SHALL être remplacé en entier, sans fusion. Un champ vide après suppression des espaces, ou une liste sans entrée non vide, SHALL équivaloir à une valeur absente. Le système SHALL NE PAS vérifier à l'enregistrement que le fichier existe : cette vérification se fait au lancement de l'étape. Ces paramètres ne SHALL pas être définissables au niveau d'un change et SHALL n'avoir d'effet que lorsque le pilote résolu est `custom`.

#### Scenario: Surcharge champ par champ
- **WHEN** la Configuration définit `uiMcpConfig` à `/etc/mcp/ui.json` et `uiAllowedTools` à `["mcp__cypress"]`, et que le workspace `A` ne surcharge que `uiAllowedTools` avec `["mcp__cypress", "mcp__db"]`
- **THEN** la résolution pour `A` donne `/etc/mcp/ui.json` et les deux outils, sans fusion avec la liste de Configuration

#### Scenario: Liste vide
- **WHEN** l'utilisateur enregistre `uiAllowedTools` avec `[]` ou `["  "]`
- **THEN** le paramètre est considéré absent à ce niveau

#### Scenario: Trop d'entrées
- **WHEN** l'utilisateur enregistre `uiAllowedTools` avec 51 entrées non vides
- **THEN** l'enregistrement est refusé avec une erreur de validation

#### Scenario: Fichier absent à l'enregistrement
- **WHEN** l'utilisateur enregistre `uiMcpConfig` avec un chemin qui n'existe pas
- **THEN** l'enregistrement réussit

#### Scenario: Pas de paramètre du pilote au niveau du change
- **WHEN** une requête de réglage de change contient `uiMcpConfig` ou `uiAllowedTools`
- **THEN** la requête est rejetée avec une erreur de validation

### Requirement: Guidage de la vérification UI
Le système SHALL permettre de définir, à la Configuration et au workspace, un guidage texte libre `uiGuidance` (Markdown, 4000 caractères au plus) destiné à l'agent de l'étape `ui`. Contrairement aux autres paramètres de lancement, les deux niveaux SHALL se cumuler et non se surcharger : le texte résolu est celui de la Configuration suivi de celui du workspace, séparés par une ligne vide, chaque texte étant d'abord débarrassé des espaces qui l'entourent. Un texte vide après suppression des espaces SHALL équivaloir à une valeur absente. Un texte de plus de 4000 caractères SHALL être rejeté avec une erreur de validation. Le guidage ne SHALL pas être définissable au niveau d'un change.

#### Scenario: Cumul des niveaux
- **WHEN** la Configuration définit `uiGuidance` à « Viser le desktop 1280 px » et le workspace à « Ignorer l'onglet Admin »
- **THEN** le guidage résolu est « Viser le desktop 1280 px », une ligne vide, puis « Ignorer l'onglet Admin »

#### Scenario: Un seul niveau
- **WHEN** seul le workspace définit un guidage
- **THEN** le guidage résolu est le texte du workspace, sans ligne vide ajoutée

#### Scenario: Texte trop long
- **WHEN** l'utilisateur enregistre un guidage de 4001 caractères
- **THEN** l'enregistrement est refusé avec une erreur de validation et le guidage existant n'est pas modifié

#### Scenario: Pas de guidage au niveau du change
- **WHEN** une requête de réglage de change contient `uiGuidance`
- **THEN** la requête est rejetée avec une erreur de validation

### Requirement: Persistance et exposition des réglages du pilote
Les réglages `uiDriver`, `uiMcpConfig`, `uiAllowedTools` et `uiGuidance` SHALL être acceptés, avec `null` pour réinitialiser un champ, par les mêmes points d'entrée que les autres réglages de vérification (`verificationDefaults` des préférences, `verification` des réglages de workspace) et persistés au même endroit (`preferences.json`, section du workspace pour la surcharge). Leur absence SHALL équivaloir à `auto`, sans liste d'outils ni guidage, sans migration ni erreur au chargement d'un fichier existant. Une section ou un champ devenu vide SHALL être retiré du fichier. La réponse de `GET /workspaces/{id}/settings` SHALL les inclure dans `overrides`, `inherited` et `resolved` ; `resolved.uiGuidance` SHALL porter le texte cumulé.

#### Scenario: Fichiers existants
- **WHEN** l'application démarre avec un `preferences.json` sans aucun de ces champs
- **THEN** le chargement réussit, le pilote résolu est `auto` et aucun guidage n'est défini

#### Scenario: Enregistrement et lecture
- **WHEN** l'utilisateur envoie `{"verification": {"uiDriver": "playwright", "uiGuidance": "Viser le desktop"}}` pour le workspace `A`
- **THEN** la section de `A` dans `preferences.json` porte ces deux champs
- **THEN** `overrides` les retourne, `inherited` porte les valeurs de Configuration et `resolved` porte le pilote et le guidage effectifs

#### Scenario: Retrait d'un champ devenu vide
- **WHEN** l'utilisateur réinitialise le dernier réglage de vérification d'un workspace qui n'a aucune autre surcharge
- **THEN** la section de ce workspace est retirée de `preferences.json`

### Requirement: Champs du pilote dans les écrans de réglage
Le sous-onglet « Vérification » de Configuration SHALL permettre de choisir le pilote parmi `auto`, `playwright` et `custom` (sans `chrome`) et de saisir `uiMcpConfig`, `uiAllowedTools` (une entrée par ligne) et `uiGuidance`. Celui de Settings SHALL proposer en plus `chrome` et, pour chaque réglage, l'héritage de la valeur de Configuration, avec la valeur héritée affichée et une surcharge distinguée d'une valeur héritée et réinitialisable ; pour `uiGuidance`, l'écran SHALL présenter le texte hérité en lecture seule au-dessus du champ du workspace puisque les deux se cumulent. Les champs `uiMcpConfig` et `uiAllowedTools` SHALL n'être actifs que lorsque le pilote résolu est `custom`. Choisir `chrome` SHALL afficher un avertissement indiquant que l'agent pilotera le navigateur de l'utilisateur avec ses sessions ouvertes. L'aide du champ `uiGuidance` SHALL demander de ne pas y écrire de mot de passe ni de secret et d'indiquer plutôt le nom de la variable d'environnement qui les porte, puisque le texte est stocké en clair et envoyé au modèle. Les valeurs SHALL être enregistrées avec les mêmes règles de validation que l'API.

#### Scenario: Configuration sans chrome
- **WHEN** l'utilisateur ouvre Configuration > Vérification
- **THEN** le sélecteur de pilote propose `auto`, `playwright` et `custom`, et pas `chrome`

#### Scenario: Settings avec chrome et avertissement
- **WHEN** l'utilisateur choisit `chrome` dans Settings > Vérification
- **THEN** un avertissement indique que l'agent pilotera le navigateur de l'utilisateur et le réglage est enregistré comme surcharge du workspace

#### Scenario: Champs custom inactifs
- **WHEN** le pilote résolu est `playwright`
- **THEN** les champs `uiMcpConfig` et `uiAllowedTools` sont désactivés

#### Scenario: Valeur héritée affichée
- **WHEN** Configuration définit le pilote `playwright` et que le workspace n'a pas de surcharge
- **THEN** Settings présente « Hérité » avec la valeur « playwright »

#### Scenario: Guidage cumulé
- **WHEN** Configuration définit un guidage et que l'utilisateur ouvre le champ de guidage d'un workspace
- **THEN** le guidage de Configuration est affiché en lecture seule au-dessus du champ du workspace

#### Scenario: Mise en garde sur les secrets
- **WHEN** l'utilisateur ouvre le champ `uiGuidance`
- **THEN** son aide demande de ne pas y écrire de mot de passe ni de secret
