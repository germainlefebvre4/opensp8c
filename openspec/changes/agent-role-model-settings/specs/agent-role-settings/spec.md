# Spec Delta

## Purpose

Permet de régler l'agent, le modèle et l'effort de raisonnement utilisés par chaque rôle d'exécution des agents (exploration, fast-forward, implémentation, correction, documentation), globalement puis par workspace, avec héritage entre niveaux et valeurs par défaut adaptées à chaque rôle.

## ADDED Requirements

### Requirement: Rôles d'exécution des agents
Le système SHALL distinguer cinq rôles d'exécution pour les subprocess d'agents CLI qu'il lance : `explorer` (sessions d'exploration nommées et anonymes), `ff` (exécutions fast-forward qui génèrent les artefacts d'un change), `implementer` (workers du pool qui implémentent les tâches d'un change), `fixer` (relancement d'un worker après « Demander des corrections » en revue HITL) et `documenter` (génération de la documentation). Chaque subprocess lancé SHALL être associé à exactement un de ces rôles.

#### Scenario: Rôle d'une session d'exploration
- **WHEN** une session d'exploration nommée ou anonyme est démarrée
- **THEN** ses réglages sont ceux du rôle `explorer`

#### Scenario: Rôle d'un fast-forward
- **WHEN** un fast-forward est déclenché pour un change
- **THEN** ses réglages sont ceux du rôle `ff`

#### Scenario: Rôle d'un worker du pool
- **WHEN** un worker du pool démarre l'implémentation d'un change de la colonne To Do
- **THEN** ses réglages sont ceux du rôle `implementer`

#### Scenario: Rôle d'une génération de documentation
- **WHEN** la génération de documentation d'un workspace est lancée
- **THEN** ses réglages sont ceux du rôle `documenter`

### Requirement: Réglage agent, modèle et effort avec héritage
Le système SHALL permettre de définir, pour un niveau de réglage, trois champs indépendants : l'agent CLI, le modèle et l'effort de raisonnement. Un champ vide SHALL signifier « hériter du niveau supérieur ». Il existe un réglage global, qui sert de base à tous les rôles, et un réglage par rôle. Le modèle et l'effort ne SHALL être hérités d'un niveau que si l'agent résolu est celui de ce niveau ; sinon ils SHALL retomber sur le préréglage de l'agent résolu, puis sur le défaut du CLI.

#### Scenario: Rôle sans réglage propre
- **WHEN** aucun champ du rôle `implementer` n'est défini et que le réglage global définit l'agent `claude`, le modèle `sonnet` et l'effort `medium`
- **THEN** un worker lancé pour l'implémentation utilise `claude`, `sonnet` et `medium`

#### Scenario: Surcharge partielle d'un rôle
- **WHEN** le rôle `documenter` ne définit que l'effort `low` et que le réglage global définit `claude` et `sonnet`
- **THEN** la génération de documentation utilise `claude`, `sonnet` et l'effort `low`

#### Scenario: Le rôle change d'agent
- **WHEN** le réglage global définit `claude` avec le modèle `sonnet` et que le rôle `ff` définit l'agent `codex` sans modèle
- **THEN** le fast-forward utilise `codex` sans hériter du modèle `sonnet`, qui n'a pas de sens pour cet agent

#### Scenario: Agent global hérité
- **WHEN** un rôle ne définit pas d'agent
- **THEN** il utilise l'agent du réglage global, lequel à défaut utilise l'agent par défaut de la plateforme

### Requirement: Résolution en cascade global puis workspace
Pour chaque lancement, le système SHALL résoudre les trois champs, un par un, en parcourant dans cet ordre le premier niveau qui définit une valeur : rôle du workspace, réglage global du workspace, rôle de Configuration, réglage global de Configuration, préréglage du rôle pour l'agent résolu, défaut du CLI (aucun flag passé). Les réglages d'un workspace SHALL n'être appliqués qu'aux subprocess lancés pour ce workspace.

#### Scenario: Le workspace surcharge Configuration
- **WHEN** le rôle `implementer` de Configuration définit `sonnet` et que le rôle `implementer` du workspace `A` définit `opus`
- **THEN** un worker du workspace `A` utilise `opus`
- **THEN** un worker du workspace `B`, qui n'a pas de surcharge, utilise `sonnet`

#### Scenario: Le global du workspace prime sur le rôle de Configuration
- **WHEN** le réglage global du workspace `A` définit le modèle `haiku` et que le rôle `explorer` de Configuration définit `opus`, sans surcharge de rôle dans le workspace
- **THEN** une exploration du workspace `A` utilise `haiku`

#### Scenario: Aucune valeur définie nulle part
- **WHEN** un subprocess est lancé pour un agent sans préréglage et qu'aucun niveau ne définit de modèle
- **THEN** aucun flag de modèle n'est passé au CLI, qui utilise son défaut

### Requirement: Préréglages par défaut des rôles pour l'agent Claude
Lorsque l'agent résolu est Claude et qu'aucun niveau ne définit un champ, le système SHALL appliquer un préréglage propre à chaque rôle : `explorer` avec un modèle de raisonnement (`opus`) et l'effort `high` ; `ff`, `implementer` et `fixer` avec un modèle de code (`sonnet`) et l'effort `medium` ; `documenter` avec un modèle économique (`haiku`) et l'effort `low`. Pour les autres agents, il n'y a pas de préréglage.

#### Scenario: Préréglage de l'explorateur avec Claude
- **WHEN** aucun réglage n'est enregistré et qu'une exploration est lancée avec l'agent `claude`
- **THEN** le subprocess reçoit le modèle `opus` et l'effort `high`

#### Scenario: Préréglage du documenteur avec Claude
- **WHEN** aucun réglage n'est enregistré et que la documentation est générée avec l'agent `claude`
- **THEN** le subprocess reçoit le modèle `haiku` et l'effort `low`

#### Scenario: Pas de préréglage pour un autre agent
- **WHEN** aucun réglage n'est enregistré et qu'un worker est lancé avec l'agent `gemini`
- **THEN** aucun flag de modèle ni d'effort n'est passé

### Requirement: Verrou d'agent de session prioritaire
Pour une session d'exploration dont l'agent est verrouillé, le système SHALL conserver cet agent quel que soit le réglage de rôle. Le modèle et l'effort de la session SHALL alors être résolus pour l'agent verrouillé selon la cascade, comme si cet agent était celui du rôle.

#### Scenario: Le rôle demande un autre agent que la session
- **WHEN** une session d'exploration existante est verrouillée sur `claude` et que le rôle `explorer` du workspace définit l'agent `codex`
- **THEN** la session continue avec `claude`
- **THEN** son modèle et son effort sont ceux définis ou préréglés pour `claude`

#### Scenario: Nouvelle session sans verrou
- **WHEN** une nouvelle session d'exploration est créée et que le rôle `explorer` définit l'agent `codex`
- **THEN** la session utilise `codex` et l'agent est verrouillé sur cette valeur pour toute sa durée

### Requirement: Portée du rôle fixer
Le rôle `fixer` SHALL s'appliquer uniquement au subprocess démarré lorsqu'un worker est relancé après « Demander des corrections » en revue HITL. La boucle d'auto-guérison qui suit un échec de validation partage le subprocess de l'implémentation et SHALL conserver les réglages du rôle `implementer`. L'interface SHALL indiquer cette portée sur le réglage du rôle `fixer`.

#### Scenario: Auto-guérison sur le modèle de l'implémenteur
- **WHEN** la validation d'un worker échoue et que la boucle d'auto-guérison réinjecte l'erreur
- **THEN** les tours de correction utilisent le modèle et l'effort du rôle `implementer`

#### Scenario: Relance après une demande de corrections
- **WHEN** l'utilisateur clique sur « Demander des corrections » et que le worker est relancé
- **THEN** le nouveau subprocess utilise les réglages du rôle `fixer`

### Requirement: Catalogue de modèles par agent
Le système SHALL exposer, pour chaque agent supporté, la liste des modèles proposables. Cette liste SHALL être construite à partir d'une liste amorcée par la plateforme (alias stables lorsque l'agent en propose : par exemple `fable`, `opus`, `sonnet`, `haiku` pour Claude et `auto`, `pro`, `flash`, `flash-lite` pour Gemini), complétée par la liste dynamique fournie par le CLI lorsque celui-ci sait lister ses modèles (Antigravity). Quand la découverte dynamique échoue, le système SHALL retomber sur la liste amorcée sans erreur bloquante. Pour tout agent, l'utilisateur SHALL pouvoir saisir librement un identifiant de modèle absent de la liste.

#### Scenario: Liste dynamique disponible
- **WHEN** l'utilisateur ouvre le sélecteur de modèle pour l'agent Antigravity et que le CLI est installé
- **THEN** les modèles retournés par le CLI sont proposés avec leur libellé

#### Scenario: Liste dynamique indisponible
- **WHEN** la découverte de modèles échoue ou que le CLI n'est pas installé
- **THEN** la liste amorcée de l'agent est proposée
- **THEN** aucune erreur bloquante n'est affichée

#### Scenario: Saisie libre
- **WHEN** l'utilisateur saisit un identifiant de modèle absent du catalogue pour l'agent `codex` et enregistre
- **THEN** cet identifiant est enregistré et utilisé tel quel au lancement

#### Scenario: Agent sans liste
- **WHEN** l'utilisateur ouvre le sélecteur de modèle pour un agent sans liste amorcée ni découverte dynamique
- **THEN** seule la saisie libre est proposée, avec le choix « défaut du CLI »

### Requirement: Niveaux d'effort déclarés par agent
Le système SHALL déclarer, pour chaque agent supporté, la liste ordonnée des niveaux d'effort qu'il accepte, vide lorsque l'agent n'en prend pas. Les niveaux SHALL être fournis par le backend et non figés dans l'interface. Le champ effort SHALL être masqué ou désactivé pour un agent qui n'en déclare aucun, et un effort enregistré SHALL ne jamais être passé à un tel agent.

#### Scenario: Niveaux de Claude
- **WHEN** l'utilisateur ouvre le champ effort pour l'agent `claude`
- **THEN** les niveaux `low`, `medium`, `high`, `xhigh` et `max` sont proposés

#### Scenario: Agent sans effort
- **WHEN** l'utilisateur sélectionne l'agent `gemini` pour un rôle
- **THEN** le champ effort n'est pas proposé pour ce rôle

#### Scenario: Effort hérité invalide pour l'agent résolu
- **WHEN** l'effort hérité (`xhigh`) n'appartient pas aux niveaux acceptés par l'agent résolu
- **THEN** l'effort n'est pas passé au CLI et le défaut de l'agent s'applique

### Requirement: Application des réglages au lancement du subprocess
Le système SHALL transmettre le modèle et l'effort résolus au CLI de l'agent au démarrage du subprocess, avec le flag propre à chaque agent, et SHALL omettre tout flag dont la valeur résolue est vide. Les réglages SHALL être résolus au moment du lancement : une modification ultérieure n'affecte pas un subprocess déjà démarré ni la session en cours, mais s'applique aux lancements suivants.

#### Scenario: Flags passés à Claude
- **WHEN** un subprocess `claude` est lancé avec le modèle `sonnet` et l'effort `medium`
- **THEN** la commande contient `--model sonnet` et `--effort medium`

#### Scenario: Effort omis pour Codex
- **WHEN** un subprocess `codex` est lancé avec un modèle défini
- **THEN** la commande contient le flag de modèle du CLI et aucun flag d'effort

#### Scenario: Agent sans flag de modèle déclaré
- **WHEN** un subprocess est lancé pour un agent dont le flag de modèle n'est pas validé par la plateforme
- **THEN** aucun flag de modèle ni d'effort n'est passé, et l'interface désactive le champ modèle de cet agent en expliquant pourquoi

#### Scenario: Modification pendant un worker actif
- **WHEN** l'utilisateur modifie le modèle du rôle `implementer` alors qu'un worker est actif
- **THEN** le worker actif conserve le modèle avec lequel il a démarré
- **THEN** le prochain worker lancé utilise le nouveau modèle

### Requirement: Validation des réglages
Le système SHALL rejeter avec une erreur de validation, sans modifier les réglages enregistrés, toute mise à jour dont l'agent n'appartient pas aux agents supportés, dont l'effort n'appartient pas aux niveaux déclarés par l'agent effectivement défini au même niveau, ou dont le modèle est vide après suppression des espaces, commence par un tiret, contient un espace ou un caractère de contrôle, ou dépasse une longueur maximale raisonnable. Le rôle ciblé SHALL appartenir aux cinq rôles définis.

#### Scenario: Agent inconnu
- **WHEN** une mise à jour fixe l'agent d'un rôle à une valeur absente des agents supportés
- **THEN** la requête est rejetée avec une erreur de validation et aucun réglage n'est modifié

#### Scenario: Modèle assimilable à un flag
- **WHEN** une mise à jour fixe un modèle à `--dangerously-skip-permissions`
- **THEN** la requête est rejetée avec une erreur de validation

#### Scenario: Rôle inconnu
- **WHEN** une mise à jour cible un rôle autre que les cinq rôles définis
- **THEN** la requête est rejetée avec une erreur de validation

#### Scenario: Effort hors niveaux de l'agent
- **WHEN** une mise à jour fixe l'agent `antigravity` et l'effort `xhigh`
- **THEN** la requête est rejetée avec une erreur de validation

### Requirement: Persistance rétrocompatible
Les réglages globaux et par rôle SHALL être persistés dans `preferences.json` et retournés par `GET /api/preferences`. L'absence de ces champs SHALL équivaloir au comportement de préréglages et de défauts décrit par cette capacité, sans migration ni erreur au chargement d'un fichier existant.

#### Scenario: Fichier de préférences existant
- **WHEN** l'application démarre avec un `preferences.json` sans réglage de rôle
- **THEN** le chargement réussit et les préréglages par défaut s'appliquent

#### Scenario: Enregistrement d'un réglage de rôle
- **WHEN** l'utilisateur enregistre le modèle `haiku` pour le rôle `documenter` dans Configuration
- **THEN** `preferences.json` reflète ce réglage et `GET /api/preferences` le retourne
- **THEN** les autres rôles restent inchangés
