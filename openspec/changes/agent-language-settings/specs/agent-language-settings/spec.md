# Spec Delta

## Purpose

Permet de régler, globalement pour la plateforme, la langue dans laquelle les agents CLI s'expriment pour les échanges (chat), pour la documentation et les artefacts OpenSpec, et pour le code qu'ils produisent, et garantit que ces langues leur sont transmises à chaque lancement.

## ADDED Requirements

### Requirement: Trois niveaux de langue indépendants
Le système SHALL gérer trois réglages de langue indépendants pour les agents : `chat` (langue des échanges avec l'agent), `documentation` (langue de la documentation générée et des artefacts OpenSpec : proposal, design, specs, tasks) et `code` (langue de tout ce qui est écrit dans le code : identifiants, commentaires, messages de commit, messages d'erreur et textes visibles). Ces réglages SHALL être globaux à la plateforme et non liés à un workspace.

#### Scenario: Valeurs par défaut
- **WHEN** aucun réglage de langue d'agent n'a jamais été enregistré
- **THEN** `chat` et `documentation` suivent la langue de l'application (`auto`)
- **THEN** `code` vaut l'anglais (`en`)

#### Scenario: Indépendance des niveaux
- **WHEN** l'utilisateur modifie le réglage `code`
- **THEN** les réglages `chat` et `documentation` restent inchangés

#### Scenario: Réglage identique pour tous les workspaces
- **WHEN** plusieurs workspaces sont configurés
- **THEN** les trois réglages s'appliquent de la même façon aux agents lancés pour chacun d'eux

### Requirement: Valeurs autorisées et extensibilité des langues
Chaque niveau SHALL accepter un code de langue appartenant à la liste des langues supportées par la plateforme. Les niveaux `chat` et `documentation` SHALL en outre accepter la valeur `auto`, qui désigne la langue courante de l'application ; le niveau `code` n'accepte pas `auto`. La liste des langues supportées SHALL être définie en un seul endroit et contenir initialement l'anglais (`en`) et le français (`fr`) ; ajouter une langue à cette liste SHALL la rendre disponible pour les trois niveaux sans autre modification.

#### Scenario: Code de langue supporté
- **WHEN** une mise à jour des préférences fixe un niveau à un code de la liste des langues supportées
- **THEN** la valeur est acceptée et enregistrée

#### Scenario: Code de langue inconnu
- **WHEN** une mise à jour des préférences fixe un niveau à un code absent de la liste des langues supportées
- **THEN** la requête est rejetée avec une erreur de validation et aucun réglage n'est modifié

#### Scenario: `auto` refusé pour le code
- **WHEN** une mise à jour des préférences fixe le niveau `code` à `auto`
- **THEN** la requête est rejetée avec une erreur de validation

#### Scenario: Ajout d'une nouvelle langue
- **WHEN** un code de langue est ajouté à la liste des langues supportées
- **THEN** il est accepté pour `chat`, `documentation` et `code`

### Requirement: Persistance et exposition via les préférences
Les trois réglages SHALL être persistés globalement dans `preferences.json` et lus/écrits via les endpoints existants `GET`/`PATCH /api/preferences`. Une mise à jour SHALL pouvoir modifier un seul niveau sans altérer les deux autres. Un `preferences.json` qui ne contient aucun de ces champs SHALL se comporter comme si les valeurs par défaut étaient enregistrées.

#### Scenario: Lecture des réglages
- **WHEN** `GET /api/preferences` est appelé
- **THEN** la réponse contient la valeur enregistrée de chacun des trois niveaux (valeur par défaut si jamais définie)

#### Scenario: Mise à jour partielle
- **WHEN** `PATCH /api/preferences` est appelé avec uniquement le niveau `documentation`
- **THEN** `preferences.json` est mis à jour pour ce niveau et les niveaux `chat` et `code` conservent leur valeur

#### Scenario: Préférences existantes sans réglage de langue
- **WHEN** l'application démarre avec un `preferences.json` antérieur à cette fonctionnalité
- **THEN** aucune erreur n'est produite et les valeurs par défaut s'appliquent

### Requirement: Synchronisation de la langue de l'interface vers le backend
Le système SHALL transmettre au backend la langue courante de l'interface au chargement de l'application et à chaque changement de langue, et la persister globalement, afin que la valeur `auto` puisse être résolue au lancement d'un agent, y compris lorsqu'aucun navigateur n'est en train d'émettre une requête (workers du pool en arrière-plan).

#### Scenario: Changement de langue de l'interface
- **WHEN** l'utilisateur bascule l'interface de l'anglais au français
- **THEN** la langue courante enregistrée côté backend devient `fr`

#### Scenario: Chargement de l'application
- **WHEN** l'application se charge avec une langue d'interface déjà mémorisée dans le navigateur
- **THEN** cette langue est transmise au backend sans action de l'utilisateur

#### Scenario: Langue d'interface jamais transmise
- **WHEN** aucune langue d'interface n'a jamais été enregistrée côté backend et qu'un niveau vaut `auto`
- **THEN** `auto` est résolu en anglais

### Requirement: Résolution des langues à chaque lancement d'agent
À chaque lancement ou reprise d'un agent, le système SHALL résoudre chaque niveau en une langue concrète : la valeur explicite si elle est définie, sinon la langue courante de l'application enregistrée côté backend pour `auto`. Un changement de réglage ou de langue de l'application SHALL s'appliquer aux lancements suivants ; il SHALL NOT modifier la consigne déjà transmise à un agent en cours d'exécution.

#### Scenario: `auto` suit la langue de l'application
- **WHEN** `chat` vaut `auto`, la langue de l'application est `fr`, et une session d'exploration est lancée
- **THEN** l'agent reçoit le français comme langue de chat

#### Scenario: Valeur explicite prioritaire
- **WHEN** `documentation` vaut `en` alors que la langue de l'application est `fr`, et une génération de documentation est lancée
- **THEN** l'agent reçoit l'anglais comme langue de documentation

#### Scenario: Changement pendant qu'un agent tourne
- **WHEN** l'utilisateur modifie un réglage de langue alors qu'un worker du pool est en cours d'exécution
- **THEN** ce worker n'est pas interrompu ni modifié, et les lancements suivants utilisent la nouvelle valeur

### Requirement: Langues transmises selon le rôle de l'agent
Le système SHALL transmettre à l'agent, à son lancement, les langues pertinentes pour son rôle : l'exploration nommée ou anonyme reçoit la langue `chat` ; la génération de documentation, le fast-forward d'un change et la promotion d'une exploration en change reçoivent la langue `documentation` ; un worker du pool qui implémente un change reçoit la langue `code` pour ce qu'il écrit dans le code et la langue `documentation` pour les fichiers OpenSpec ou de documentation qu'il modifie.

#### Scenario: Exploration
- **WHEN** une session d'exploration nommée ou anonyme est lancée
- **THEN** l'agent reçoit la langue `chat` résolue

#### Scenario: Fast-forward et promotion
- **WHEN** un fast-forward est lancé sur un change, ou qu'une exploration est promue en change
- **THEN** l'agent reçoit la langue `documentation` résolue pour la rédaction des artefacts

#### Scenario: Génération de documentation
- **WHEN** une génération de documentation est lancée
- **THEN** l'agent reçoit la langue `documentation` résolue

#### Scenario: Worker du pool
- **WHEN** un worker du pool est lancé pour implémenter un change
- **THEN** l'agent reçoit la langue `code` résolue pour le code produit et la langue `documentation` résolue pour les fichiers OpenSpec ou de documentation qu'il modifie

### Requirement: La consigne de langue est une valeur par défaut
La consigne de langue transmise à l'agent SHALL être formulée comme une valeur par défaut et non comme une contrainte ferme : une demande explicite de l'utilisateur, le contenu des artefacts du change ou les conventions du projet cible (par exemple un mécanisme d'internationalisation existant) SHALL l'emporter sur elle. Pour le niveau `code`, écrire dans une autre langue que celle du réglage SHALL nécessiter une demande explicite ou l'usage de l'internationalisation du projet.

#### Scenario: Demande explicite de l'utilisateur
- **WHEN** `chat` vaut `en` et l'utilisateur demande explicitement à l'agent de répondre en français
- **THEN** l'agent peut répondre en français sans être en contradiction avec la consigne reçue

#### Scenario: Projet cible internationalisé
- **WHEN** `code` vaut `en` et le projet cible impose que les textes visibles passent par des clés de traduction en plusieurs langues
- **THEN** la consigne reçue n'interdit pas à l'agent de suivre cette convention

#### Scenario: Périmètre du niveau code
- **WHEN** l'agent reçoit la langue `code`
- **THEN** la consigne désigne les identifiants, commentaires, messages de commit, messages d'erreur et textes visibles comme relevant de cette langue par défaut

### Requirement: Le formalisme des artefacts OpenSpec est préservé
La langue `documentation` SHALL s'appliquer à la prose des artefacts OpenSpec et de la documentation, et SHALL NOT altérer leur formalisme : les titres structurels requis (`### Requirement:`, `#### Scenario:`), les mots-clés normatifs (`SHALL`, `MUST`) et les marqueurs `WHEN`/`THEN`, ainsi que les noms de fichiers imposés, restent inchangés quelle que soit la langue.

#### Scenario: Artefact rédigé en français
- **WHEN** la langue `documentation` résolue est le français et un agent rédige une spec
- **THEN** la consigne indique que la prose est en français et que les titres structurels, `SHALL`/`MUST` et `WHEN`/`THEN` restent en anglais

#### Scenario: Noms de fichiers
- **WHEN** la langue `documentation` résolue est autre que l'anglais
- **THEN** la consigne précise que les noms de fichiers imposés ne sont pas traduits

### Requirement: Transmission indépendante du mécanisme d'injection de l'agent
Le système SHALL transmettre la consigne de langue à tous les agents supportés, y compris ceux qui ne disposent pas d'un mécanisme de prompt système : pour ces derniers, la consigne SHALL être transmise avec le message envoyé à l'agent.

#### Scenario: Agent avec prompt système
- **WHEN** un agent acceptant un prompt système supplémentaire est lancé
- **THEN** la consigne de langue lui est transmise par ce prompt

#### Scenario: Agent sans prompt système
- **WHEN** un agent sans mécanisme de prompt système supplémentaire est lancé
- **THEN** la consigne de langue lui est tout de même transmise, avec son message

### Requirement: Le lancement d'un agent ne dépend pas de la lisibilité des réglages de langue
Si les réglages de langue ne peuvent pas être lus au moment d'un lancement, le système SHALL lancer l'agent avec les valeurs par défaut plutôt que de faire échouer le lancement.

#### Scenario: Préférences illisibles
- **WHEN** les préférences ne peuvent pas être chargées lors du lancement d'un agent
- **THEN** l'agent est lancé avec les langues par défaut (`chat` et `documentation` en anglais faute de langue d'application connue, `code` en anglais)
