# Spec Delta

## MODIFIED Requirements

### Requirement: Scénarios et tâches soumis à l'agent
Une fois l'application prête, la plateforme SHALL envoyer à l'agent un tour contenant l'URL de base, le dossier de preuves (variable `OPENSP8C_VERIFY_ARTIFACTS`), la liste de ce qu'il doit vérifier : les scénarios (`#### Scenario:` avec leurs lignes WHEN et THEN) des fichiers `openspec/changes/<change>/specs/**/spec.md` de la branche, et le texte, marqueur retiré, de chaque tâche non cochée portant le marqueur `<!-- human review required -->` du `tasks.md` de la branche, puis le guidage de l'utilisateur (`uiGuidance` résolu, voir `verification-settings`) lorsqu'il n'est pas vide. La consigne système SHALL imposer la lecture seule des fichiers du dépôt, et lui demander de n'exécuter que ce qui est observable dans l'interface, de déposer ses captures dans le dossier de preuves et de conclure selon le contrat de verdict. Pour le pilote `auto`, elle SHALL laisser l'agent choisir l'outil de navigation parmi ceux dont il dispose ; pour tout autre pilote, elle SHALL lui imposer de ne naviguer qu'avec les outils de ce pilote, sans lancer de script jetable. Le guidage SHALL être présenté comme des indications de l'utilisateur qui ne dispensent jamais de la lecture seule ni du contrat de verdict.

#### Scenario: Contenu du tour
- **WHEN** un change a deux scénarios de spec et une tâche marquée non cochée « 4.2 Parcours manuel de la navigation »
- **THEN** le tour envoyé à l'agent contient les deux scénarios, le texte « 4.2 Parcours manuel de la navigation » sans le marqueur, l'URL de base et le chemin du dossier de preuves

#### Scenario: Tâches déjà cochées
- **WHEN** une tâche marquée est déjà cochée
- **THEN** elle n'est pas soumise à l'agent

#### Scenario: Change sans spec delta
- **WHEN** le change n'a aucun fichier de spec delta et aucune tâche marquée
- **THEN** le tour est envoyé sans liste de scénarios et l'agent est invité à répondre `VERDICT: SKIP` s'il ne peut rien observer

#### Scenario: Guidage dans le tour
- **WHEN** le guidage résolu est « Viser le desktop 1280 px »
- **THEN** le tour contient un bloc d'indications de l'utilisateur avec ce texte, après la liste de ce qu'il faut vérifier

#### Scenario: Pas de guidage
- **WHEN** le guidage résolu est vide
- **THEN** le tour ne contient aucun bloc d'indications de l'utilisateur

#### Scenario: Consigne en pilote auto
- **WHEN** le pilote résolu est `auto`
- **THEN** la consigne système laisse l'agent choisir l'outil de navigation parmi ceux dont il dispose

#### Scenario: Consigne en pilote imposé
- **WHEN** le pilote résolu est `playwright`
- **THEN** la consigne système impose de ne naviguer qu'avec les outils du pilote et interdit un script jetable

## ADDED Requirements

### Requirement: Sélection du pilote et lancement de l'agent
L'étape `ui` SHALL lancer l'agent de rôle `verifier` selon le pilote résolu (voir `verification-settings`) en ajoutant à ses paramètres de lancement ceux du pilote, portés par le champ `ExtraArgs` de la configuration de l'agent, sans modifier les paramètres des autres lancements d'agent. Pour `auto`, aucun paramètre propre au pilote ne SHALL être ajouté. Pour `playwright`, la plateforme SHALL fournir une configuration MCP à un seul serveur nommé `playwright` (navigateur headless, profil isolé en mémoire, dossier de sortie égal au dossier de preuves du run, version du paquet fixée par la plateforme et non `latest`), la passer avec `--mcp-config` et `--strict-mcp-config` pour qu'aucun serveur de l'hôte ne s'y ajoute, et autoriser par `--allowedTools` les outils de ce serveur ainsi que `Read`, `Grep` et `Glob`. Pour `chrome`, la plateforme SHALL passer `--chrome` et autoriser les outils de l'intégration Chrome ainsi que `Read`, `Grep` et `Glob`. Pour `custom`, la plateforme SHALL passer `uiMcpConfig` avec `--mcp-config` et `--strict-mcp-config`, et autoriser les `uiAllowedTools` ainsi que `Read`, `Grep` et `Glob`. La configuration MCP générée pour `playwright` SHALL être écrite en dehors du worktree et supprimée à la fin de l'étape, quelle qu'en soit l'issue.

#### Scenario: Pilote auto
- **WHEN** le pilote résolu est `auto`
- **THEN** l'agent est lancé sans `--mcp-config`, sans `--allowedTools` et sans `--chrome`

#### Scenario: Pilote playwright
- **WHEN** le pilote résolu est `playwright`
- **THEN** l'agent est lancé avec `--mcp-config` vers un fichier déclarant le seul serveur `playwright`, `--strict-mcp-config` et `--allowedTools` listant les outils de ce serveur, `Read`, `Grep` et `Glob`

#### Scenario: Captures dans le dossier de preuves
- **WHEN** le pilote est `playwright`
- **THEN** le dossier de sortie du serveur est le dossier de preuves du run et les captures y apparaissent dans la liste des preuves du rapport

#### Scenario: Pilote chrome
- **WHEN** le pilote résolu est `chrome`
- **THEN** l'agent est lancé avec `--chrome` et `--allowedTools` listant les outils de l'intégration Chrome, `Read`, `Grep` et `Glob`

#### Scenario: Pilote custom
- **WHEN** le pilote résolu est `custom` avec `uiMcpConfig` à `/etc/mcp/ui.json` et `uiAllowedTools` à `["mcp__cypress"]`
- **THEN** l'agent est lancé avec `--mcp-config /etc/mcp/ui.json`, `--strict-mcp-config` et `--allowedTools` listant `mcp__cypress`, `Read`, `Grep` et `Glob`

#### Scenario: Fichier de configuration temporaire supprimé
- **WHEN** l'étape `ui` en pilote `playwright` se termine, par réussite, échec ou annulation
- **THEN** le fichier de configuration MCP généré n'existe plus et aucun fichier n'a été créé dans le worktree

### Requirement: Pilote réellement disponible
Pour un pilote autre que `auto`, l'étape `ui` SHALL vérifier la disponibilité du pilote avant de lancer l'application, puis dès l'événement d'initialisation du flux de l'agent. Avant le lancement de l'application, elle SHALL échouer, avec une raison désignant la cause, dans les cas suivants : le pilote `playwright` alors que `npx` est introuvable sur l'hôte ; le pilote `custom` sans `uiMcpConfig`, sans `uiAllowedTools`, ou avec un fichier absent, illisible ou qui ne contient pas de `mcpServers` ; l'agent de rôle `verifier` autre que Claude. Dès l'événement d'initialisation de l'agent, elle SHALL échouer et arrêter l'agent, avant qu'il n'ait exécuté un outil, si cet événement ne liste pas comme `connected` chaque serveur MCP attendu (`playwright`, ou ceux du fichier `custom`), si le pilote `chrome` n'y liste aucun outil de l'intégration Chrome, ou si l'événement n'a pas la forme attendue. Une raison relative à `chrome` SHALL suggérer de vérifier que Chrome est ouvert avec l'extension connectée.

#### Scenario: npx introuvable
- **WHEN** le pilote est `playwright` et que `npx` n'est pas dans le `PATH`
- **THEN** l'étape échoue avec une raison mentionnant `npx` sans avoir lancé l'application ni l'agent

#### Scenario: Fichier custom absent
- **WHEN** le pilote est `custom` et que `uiMcpConfig` désigne un fichier qui n'existe pas
- **THEN** l'étape échoue avec une raison mentionnant ce chemin sans avoir lancé l'application ni l'agent

#### Scenario: Outils autorisés manquants
- **WHEN** le pilote est `custom` sans `uiAllowedTools`
- **THEN** l'étape échoue avec la raison « uiAllowedTools non configuré »

#### Scenario: Agent non compatible
- **WHEN** le pilote est `playwright` et que le rôle `verifier` est configuré avec un agent autre que Claude
- **THEN** l'étape échoue avec une raison indiquant que ce pilote n'est supporté que par Claude, sans avoir lancé l'application

#### Scenario: Serveur MCP non connecté
- **WHEN** l'événement d'initialisation de l'agent liste le serveur `playwright` avec un statut autre que `connected`
- **THEN** l'étape échoue avec ce statut dans la raison, l'agent est arrêté et aucun outil n'a été exécuté

#### Scenario: Événement d'initialisation illisible
- **WHEN** l'événement d'initialisation ne contient pas la liste des serveurs ou des outils attendue par le pilote
- **THEN** l'étape échoue avec la raison « événement d'initialisation illisible » plutôt que de poursuivre sans contrôle

#### Scenario: Chrome indisponible
- **WHEN** le pilote est `chrome` et que l'événement d'initialisation ne liste aucun outil de l'intégration Chrome
- **THEN** l'étape échoue avec une raison suggérant de vérifier que Chrome est ouvert avec l'extension connectée, et l'agent est arrêté

#### Scenario: Pilote auto non contrôlé
- **WHEN** le pilote est `auto`
- **THEN** aucune de ces vérifications n'est appliquée

### Requirement: Aucune demande de permission en attente
L'étape `ui` SHALL lancer l'agent de façon qu'aucune demande de permission ne reste sans réponse : un outil qui ne figure pas parmi ceux autorisés par les réglages de l'hôte ou par `--allowedTools` SHALL être refusé automatiquement (`--permission-prompts none`) plutôt que d'attendre un utilisateur. Un tel refus SHALL apparaître dans le journal du run et, lorsque l'agent conclut qu'il n'a pas pu vérifier, dans son rapport. L'étape ne SHALL jamais rester bloquée jusqu'au délai maximal d'étape du seul fait d'une permission non accordée.

#### Scenario: Outil non autorisé
- **WHEN** l'agent tente d'utiliser un outil qui n'est autorisé ni par l'hôte ni par `--allowedTools`
- **THEN** l'appel est refusé sans attente et l'agent poursuit son tour

#### Scenario: Pilote auto
- **WHEN** le pilote est `auto` et que l'hôte n'autorise aucun outil de navigation
- **THEN** l'agent ne peut pas naviguer, conclut selon le contrat de verdict et l'étape ne reste pas bloquée

### Requirement: Un `PASS` exige un usage du pilote
Pour un pilote autre que `auto`, un verdict `PASS` SHALL n'être accepté que si l'agent a effectué au moins un appel d'un outil du pilote (outils du serveur `playwright`, outils de l'intégration Chrome, ou outils `mcp__*` du fichier `custom`) pendant son tour ; sinon l'étape SHALL échouer avec la raison « aucun outil du pilote n'a été utilisé », le rapport de l'agent étant conservé, et aucune tâche ne SHALL être cochée. Un verdict `FAIL`, `SKIP` ou absent SHALL être traité comme avant, sans cette exigence.

#### Scenario: PASS avec usage
- **WHEN** le pilote est `playwright` et que l'agent appelle au moins un outil du serveur `playwright` puis conclut `VERDICT: PASS`
- **THEN** le verdict est accepté

#### Scenario: PASS sans usage
- **WHEN** le pilote est `playwright` et que l'agent conclut `VERDICT: PASS` sans avoir appelé aucun outil du serveur `playwright`
- **THEN** l'étape échoue avec la raison « aucun outil du pilote n'a été utilisé » et aucune tâche n'est cochée

#### Scenario: SKIP sans usage
- **WHEN** le pilote est `chrome` et que l'agent conclut `VERDICT: SKIP` sans appel d'outil, sur un change sans tâche marquée
- **THEN** le verdict est accepté comme pour tout `SKIP`

#### Scenario: Pilote auto
- **WHEN** le pilote est `auto` et que l'agent conclut `VERDICT: PASS` sans appel d'outil `mcp__*`
- **THEN** le verdict est accepté

### Requirement: Prudence propre au pilote `chrome`
Pour le pilote `chrome`, la consigne système SHALL demander à l'agent de n'ouvrir que ses propres onglets et de ne lire, ne naviguer ni ne modifier aucun onglet existant de l'utilisateur. L'exclusivité de l'étape `ui` (voir « Verrou exclusif de la vérification UI ») SHALL continuer de s'appliquer, de sorte qu'un seul agent pilote le navigateur à la fois. Les captures prises par l'intégration Chrome n'étant pas nécessairement écrites dans le dossier de preuves, l'absence de preuve SHALL NE PAS être une cause d'échec ; le rapport de l'agent en tient lieu.

#### Scenario: Consigne sur les onglets
- **WHEN** le pilote est `chrome`
- **THEN** la consigne système demande de n'ouvrir que des onglets propres à l'agent et de laisser les onglets existants intacts

#### Scenario: Pas de capture
- **WHEN** l'agent en pilote `chrome` conclut `VERDICT: PASS` sans qu'aucun fichier n'ait été déposé dans le dossier de preuves
- **THEN** le verdict est accepté et le rapport ne liste aucune preuve

### Requirement: Pilote indiqué dans le rapport
`GET …/verification/report`, pour un run de l'étape `ui`, SHALL inclure le pilote utilisé (`driver`, l'une des valeurs `auto`, `playwright`, `chrome`, `custom`) tel qu'il a été résolu au lancement de l'étape, ainsi que la liste des outils d'`--allowedTools` passés à l'agent. Un rapport d'une autre étape, ou d'un run antérieur à ce champ, SHALL ne pas porter `driver`.

#### Scenario: Pilote dans le rapport
- **WHEN** l'étape `ui` s'est exécutée avec le pilote `playwright`
- **THEN** le rapport du run contient `driver` à `playwright` et la liste des outils autorisés

#### Scenario: Run antérieur
- **WHEN** le rapport demandé concerne un run `verify` écrit avant l'introduction des pilotes
- **THEN** la réponse est valide et ne contient pas `driver`

#### Scenario: Étape de conformité
- **WHEN** le rapport demandé est celui de l'étape `conformity`
- **THEN** la réponse ne contient pas `driver`
