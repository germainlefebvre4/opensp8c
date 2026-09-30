# workspace-settings Specification

## Purpose
Fournit un écran Settings propre au workspace actif, qui surcharge la configuration globale de la plateforme (agent pool, agent, modèle et effort par rôle, variables d'environnement) pour ce seul workspace.

## Requirements

### Requirement: Écran Settings scopé au workspace actif
L'écran Settings SHALL être lié au workspace actuellement sélectionné, comme les onglets Kanban, Specs, Timeline et Agents, et SHALL afficher le nom du workspace concerné. Lorsqu'aucun workspace n'est sélectionné, il SHALL afficher le même état vide « aucun workspace » que les autres onglets liés au workspace.

#### Scenario: Settings suit le workspace sélectionné
- **WHEN** l'utilisateur ouvre Settings sur le workspace `A` puis sélectionne le workspace `B` dans la sidebar
- **THEN** Settings affiche les surcharges du workspace `B` et plus celles de `A`

#### Scenario: Aucun workspace sélectionné
- **WHEN** aucun workspace n'est configuré ou sélectionné et que l'utilisateur ouvre Settings
- **THEN** l'application affiche l'état vide « aucun workspace »

#### Scenario: URL porteuse du workspace
- **WHEN** l'utilisateur navigue vers Settings
- **THEN** l'URL porte le workspace actif, comme pour les onglets Kanban, Specs, Timeline et Agents

### Requirement: Sous-onglets de Settings
L'écran Settings SHALL exposer quatre sous-onglets : « Agent Pool », « Colonnes » (« Columns » en anglais), « Environnement » (« Environment ») et « Spécialisations » (« Specializations »). Le sous-onglet Agent Pool SHALL proposer les mêmes réglages que celui de Configuration, avec la même vocation, mais valables pour ce workspace seulement.

#### Scenario: Affichage des sous-onglets
- **WHEN** l'utilisateur ouvre Settings avec un workspace sélectionné
- **THEN** les sous-onglets Agent Pool, Colonnes, Environnement et Spécialisations sont visibles

#### Scenario: Libellés traduits
- **WHEN** la langue de l'application est l'anglais
- **THEN** le sous-onglet « Colonnes » est libellé « Columns » et « Environnement » est libellé « Environment »

### Requirement: Surcharge et héritage visibles
Pour chaque réglage surchargeable, Settings SHALL afficher la valeur héritée de Configuration lorsque le workspace ne définit pas de surcharge, et distinguer visuellement une surcharge du workspace d'une valeur héritée. Chaque surcharge SHALL pouvoir être réinitialisée pour revenir à l'héritage de Configuration.

#### Scenario: Valeur héritée affichée
- **WHEN** le workspace n'a aucune surcharge pour le rôle `implementer` et que Configuration définit `sonnet`
- **THEN** le champ modèle de ce rôle affiche `sonnet` comme valeur héritée de Configuration

#### Scenario: Surcharge marquée
- **WHEN** l'utilisateur définit `opus` comme modèle du rôle `implementer` pour le workspace
- **THEN** le champ est présenté comme une surcharge du workspace

#### Scenario: Réinitialisation d'une surcharge
- **WHEN** l'utilisateur réinitialise la surcharge du modèle du rôle `implementer`
- **THEN** la surcharge disparaît de `preferences.json` pour ce workspace et le champ affiche à nouveau la valeur héritée

### Requirement: Surcharge de l'Agent Pool par workspace
Le sous-onglet Agent Pool de Settings SHALL permettre de surcharger, pour le workspace, la taille du pool (de 1 à 5), le mode de délégation (`full-autonomy` ou `hitl-review`) et le nombre maximal de tentatives d'auto-correction. Les valeurs non surchargées SHALL suivre les défauts de Configuration.

#### Scenario: Surcharge de la taille du pool
- **WHEN** l'utilisateur définit la taille `4` dans Settings > Agent Pool du workspace `A`
- **THEN** le pool démarré pour `A` a une taille de 4
- **THEN** le pool d'un autre workspace garde la valeur par défaut de Configuration

#### Scenario: Valeur invalide
- **WHEN** l'utilisateur tente d'enregistrer une taille de pool de 9
- **THEN** l'enregistrement est refusé avec une erreur de validation et la surcharge existante n'est pas modifiée

### Requirement: Réglage des rôles par workspace dans Colonnes
Le sous-onglet Colonnes de Settings SHALL afficher un réglage global du workspace et un réglage pour chacun des cinq rôles (`explorer`, `ff`, `implementer`, `fixer`, `documenter`), chacun avec ses champs agent, modèle et effort, selon les règles de `agent-role-settings`. Chaque ligne SHALL indiquer la colonne ou l'étape du Kanban à laquelle son rôle correspond.

#### Scenario: Affichage des lignes de rôle
- **WHEN** l'utilisateur ouvre Settings > Colonnes
- **THEN** une ligne globale et cinq lignes de rôle sont affichées, chacune avec un sélecteur d'agent, un sélecteur de modèle avec saisie libre et, lorsque l'agent en propose, un sélecteur d'effort

#### Scenario: Enregistrement d'une surcharge de rôle
- **WHEN** l'utilisateur choisit le modèle `haiku` pour le rôle `documenter` et enregistre
- **THEN** la surcharge est persistée dans la section du workspace et s'applique aux prochains lancements de ce workspace

### Requirement: Variables d'environnement surchargées par workspace
Le sous-onglet Environnement de Settings SHALL permettre d'ajouter et de retirer, pour le workspace, des variables d'environnement globales et des variables propres à chaque agent CLI. Au démarrage d'un subprocess pour ce workspace, le système SHALL combiner dans l'ordre : variables globales de Configuration, variables de l'agent de Configuration, variables globales du workspace, variables de l'agent du workspace ; la dernière valeur définie l'emporte.

#### Scenario: La variable du workspace surcharge Configuration
- **WHEN** Configuration définit `FOO=global` et que le workspace `A` définit `FOO=workspace`
- **THEN** un subprocess lancé pour `A` reçoit `FOO=workspace`
- **THEN** un subprocess lancé pour `B` reçoit `FOO=global`

#### Scenario: Variable d'agent du workspace prioritaire
- **WHEN** le workspace `A` définit `FOO=ws-global` et `FOO=ws-claude` pour l'agent `claude`
- **THEN** un subprocess `claude` de `A` reçoit `FOO=ws-claude`

#### Scenario: Suppression d'une variable du workspace
- **WHEN** l'utilisateur retire une variable surchargée du workspace et enregistre
- **THEN** cette variable n'est plus dans la section du workspace et la valeur de Configuration s'applique de nouveau

### Requirement: Sous-onglet Spécialisations inchangé
Le sous-onglet Spécialisations SHALL conserver le comportement actuel de gestion des tags de spécialisation d'agent : liste de base en lecture seule et gestion des tags personnalisés, dont le vocabulaire reste global à la plateforme.

#### Scenario: Gestion des tags depuis le sous-onglet
- **WHEN** l'utilisateur ajoute un tag valide dans Settings > Spécialisations
- **THEN** le tag est enregistré dans le vocabulaire global et est disponible pour tous les workspaces

### Requirement: Stockage des surcharges par workspace
Les surcharges d'un workspace SHALL être persistées dans `preferences.json` dans une section indexée par l'identifiant stable du workspace. L'absence de section SHALL équivaloir à l'absence de toute surcharge. Les sections dont l'identifiant ne correspond à aucun workspace configuré SHALL être ignorées à l'exécution sans erreur.

#### Scenario: Workspace sans surcharge
- **WHEN** un workspace n'a aucune section dans `preferences.json`
- **THEN** tous ses lancements utilisent les valeurs de Configuration

#### Scenario: Workspace retiré de la configuration
- **WHEN** un workspace est retiré de `config.yaml` alors que `preferences.json` contient sa section
- **THEN** l'application démarre normalement et ignore cette section

### Requirement: Surcharge de la commande de validation dans Settings
Le sous-onglet Agent Pool de Settings SHALL permettre de surcharger, pour le workspace, la commande de validation exécutée par les workers du pool après chaque invocation de l'agent (`validationCommand`, texte libre). Lorsque le workspace ne définit pas de surcharge, le champ SHALL afficher la valeur héritée de Configuration, ou l'indication que la commande est auto-détectée si Configuration n'en définit pas. La surcharge SHALL pouvoir être réinitialisée pour revenir à l'héritage, et SHALL être persistée dans la section du workspace de `preferences.json` avec les autres surcharges du pool.

#### Scenario: Valeur héritée affichée
- **WHEN** Configuration ne définit pas de commande de validation et que le workspace n'a aucune surcharge
- **THEN** le champ indique que la commande est auto-détectée, sans valeur de surcharge

#### Scenario: Surcharge de la commande
- **WHEN** l'utilisateur saisit `cd backend && go test ./...` comme commande de validation du workspace et enregistre
- **THEN** la surcharge est persistée pour ce workspace, distinguée visuellement d'une valeur héritée, et appliquée aux workers de ce workspace uniquement

#### Scenario: Réinitialisation
- **WHEN** l'utilisateur réinitialise la surcharge de la commande de validation
- **THEN** le workspace hérite à nouveau de Configuration, ou de l'auto-détection si Configuration ne définit rien
