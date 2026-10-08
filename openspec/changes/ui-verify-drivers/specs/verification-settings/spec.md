# Spec Delta

## ADDED Requirements

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
