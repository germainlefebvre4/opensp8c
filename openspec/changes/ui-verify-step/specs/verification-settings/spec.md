# Spec Delta

## MODIFIED Requirements

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
