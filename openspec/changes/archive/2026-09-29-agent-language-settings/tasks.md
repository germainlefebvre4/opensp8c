# Tasks

## 1. Registre des langues

- [x] 1.1 Créer le paquet `backend/internal/language` avec le registre des langues supportées (code, libellé natif, nom anglais pour la consigne), initialisé à `en` et `fr`, avec des fonctions de recherche par code et de liste ordonnée. Vérifier par un test unitaire couvrant la recherche d'un code connu, d'un code inconnu, et l'ordre stable de la liste.

## 2. Préférences : modèle, résolution et API

- [x] 2.1 Ajouter à `Preferences` les champs optionnels `AgentLanguages` (`chat`, `documentation`, `code`) et `UILocale`, avec leurs setters (mise à jour partielle d'un seul niveau sans toucher aux autres). Vérifier par des tests de `internal/preferences` : un `preferences.json` sans ces champs se charge sans erreur, et modifier `documentation` laisse `chat` et `code` intacts.
- [x] 2.2 Implémenter `language.Resolve(langs, uiLocale)` (valeurs simples, sans dépendre du paquet `preferences`) : valeur explicite, sinon `UILocale` pour `auto`, sinon `en` ; défauts `auto`/`auto`/`en` quand rien n'est enregistré ; défauts complets si les préférences sont indisponibles. Vérifier par des tests table : chat `auto` + `uiLocale=fr` donne `fr`, documentation `en` + `uiLocale=fr` donne `en`, `uiLocale` absent donne `en`, préférences illisibles donnent les défauts.
- [x] 2.3 Étendre `GET /api/preferences` (`handlers/preferences.go`) pour retourner `agentLanguages` (valeurs enregistrées ou défauts), `uiLocale` et `supportedLanguages`. Vérifier par un test de handler sur les trois champs, avec et sans réglage enregistré.
- [x] 2.4 Étendre `PATCH /api/preferences` pour accepter `agentLanguages` (partiel) et `uiLocale`, avec validation : code absent du registre refusé (400), `auto` refusé pour `code`, `uiLocale` hors registre refusé, aucune modification en cas de rejet. Vérifier par des tests de handler pour chaque cas de rejet et pour la mise à jour partielle nominale.

## 3. Consigne de langue

- [x] 3.1 Implémenter `language.Directive(role, resolved)` pour les rôles `Chat`, `Docs` et `Worker`, formulée comme valeur par défaut (l'explicite, les artefacts du change et les conventions du projet, dont l'i18n, l'emportent). Le rôle `Docs`/`Worker` précise que la langue s'applique à la prose et que les titres structurels OpenSpec, `SHALL`/`MUST`, `WHEN`/`THEN` et les noms de fichiers imposés restent inchangés ; le niveau `code` énumère identifiants, commentaires, messages de commit, messages d'erreur et textes visibles. Vérifier par des tests unitaires sur le contenu produit pour chaque rôle et pour `en` et `fr`.
- [x] 3.2 Ajouter la méthode `(*Preferences).LanguageDirective(role)`, sûre sur un receveur `nil` (chargement échoué), qui résout les langues puis construit la consigne, sur le modèle d'`EnvFor`. Vérifier par des tests : receveur `nil` retourne la consigne par défaut sans panique, et des préférences renseignées retournent la consigne résolue du rôle demandé.

## 4. Transmission par agent

- [x] 4.1 Ajouter le paramètre `languageDirective` à `session.StartSubprocess` et au seam `startSubprocessFn` du pool, et mettre à jour tous les appelants (sans encore renseigner la consigne) et les stubs de test. Vérifier par `go build ./...` et `go test ./internal/...` sans régression.
- [x] 4.2 Claude, Codex et Copilot : concaténer la consigne à `extraSystemPrompt` avant `BuildSubprocessArgs`. Vérifier par un test de construction d'arguments : la consigne apparaît dans le prompt système supplémentaire, à un lancement neuf comme à une reprise.
- [x] 4.3 Antigravity : inclure la consigne dans le cadrage du premier message et l'envoyer seule à la reprise (aujourd'hui le cadrage est supprimé à la reprise). Vérifier par des tests du writer Antigravity : premier message avec cadrage et consigne, reprise avec la consigne seule.
- [x] 4.4 Gemini : ajouter la consigne à chaque tour dans un paragraphe séparé qui ne s'insère pas dans les arguments d'une slash-command (`/opsx:ff <change>`, `/opsx:apply <change>`). Vérifier par des tests du pont Gemini : un prompt libre et un prompt commençant par `/` reçoivent la consigne, et la première ligne de ce dernier reste la commande intacte.

## 5. Branchement sur les sites de lancement

- [x] 5.1 `session/manager.go` : passer la consigne `Chat` pour l'exploration nommée et l'exploration anonyme, en réutilisant la lecture de préférences existante. Vérifier par des tests du manager avec un `StartSubprocess` observé : la langue chat résolue est transmise dans les deux chemins.
- [x] 5.2 `handlers/ff.go` et `handlers/explore.go` (promotion d'une exploration) : passer la consigne `Docs`. Vérifier par des tests de handler avec le lancement observé : la langue documentation résolue est transmise dans les deux chemins.
- [x] 5.3 `handlers/docs.go` : passer la consigne `Docs` à la génération de documentation. Vérifier par un test de `docs_test.go` : la consigne accompagne le prompt de formalisme et le nom des pages reste inchangé.
- [x] 5.4 `pool/worker.go` : passer la consigne `Worker` (langue code + langue documentation) au lancement du worker. Vérifier par un test du worker via `startSubprocessFn` : la consigne reçue mentionne les deux langues résolues.
- [x] 5.5 Vérifier qu'un changement de réglage ou de `uiLocale` s'applique au lancement suivant sans toucher à un agent en cours d'exécution. Vérifier par un test : un worker lancé avant le changement conserve sa consigne, un worker lancé après reçoit la nouvelle.

## 6. Frontend : synchronisation et types

- [x] 6.1 Étendre `Preferences` dans `frontend/src/lib/api.ts` (`agentLanguages`, `uiLocale`, `supportedLanguages`) et les hooks de `useAgentPreferences.ts` si nécessaire. Vérifier par `npm run build` sans erreur de type dans `frontend/`.
- [x] 6.2 Créer un hook monté à la racine de l'application qui envoie `uiLocale` au backend au chargement puis à chaque changement de langue de l'interface, sans importer le client HTTP dans `i18n.ts`. Vérifier par un test du hook : un appel `PATCH` au montage avec la langue courante, puis un second après un changement de langue.

## 7. Frontend : onglet Langue

- [x] 7.1 Dans `ConfigurationPage.tsx`, remplacer le `<LanguageSwitcher />` nu de l'onglet `language` par un composant d'onglet qui le conserve en tête et ajoute les trois réglages (chat, documentation, code) alimentés par `supportedLanguages` (tests à ajouter dans `ConfigurationPage.test.tsx`, sur son motif `renderToStaticMarkup` existant) ; chat et documentation ajoutent l'option `auto` qui affiche la langue résolue, code n'a pas `auto`. Enregistrer via `usePatchPreferences`. Vérifier par des tests de rendu : défauts affichés, absence d'`auto` pour le code, ajout d'une langue dans `supportedLanguages` visible dans les trois listes sans autre modification.
- [x] 7.2 Ajouter les clés de traduction correspondantes dans `frontend/src/locales/en/configuration.json` et `frontend/src/locales/fr/configuration.json`, sans clé manquante ni `defaultValue` unique. Vérifier par les tests de cohérence i18n existants (`locales/*.i18n.test.ts`) via `npm test` dans `frontend/`.
- [x] 7.3 Vérifier l'enregistrement d'un réglage depuis l'onglet : modifier la langue de la documentation, enregistrer, recharger. Vérifier par un test du composant sur l'appel `PATCH` partiel, et manuellement que la valeur persiste après rechargement.

## 8. Vérification d'intégration

- [x] 8.1 Vérifier sur les CLI réellement installées (Gemini et Antigravity en priorité) que la consigne ajoutée à un prompt slash-command ne corrompt pas l'argument de la commande (`/opsx:ff <change>`, `/opsx:apply <change>`), et consigner le résultat ; en cas de corruption, ajuster le canal de transmission décrit dans `design.md`. Vérifier par un lancement manuel de chaque agent installé. Résultat (2026-09-29) : Antigravity 1.2.13, argument intact. Gemini 0.52.0 : intact pour les commandes `opsx` livrées (aucun `{{args}}`), mais une commande custom utilisant `{{args}}` reçoit la consigne dans l'argument ; aucun ajustement du canal nécessaire pour les commandes du projet.
- [x] 8.2 Parcours de bout en bout dans l'application : régler chat sur `auto`, documentation sur `fr`, code sur `en` ; basculer l'interface en français ; lancer une exploration, un fast-forward et une génération de documentation ; confirmer que la langue observée dans les réponses et artefacts suit les réglages, et qu'une demande explicite de l'utilisateur dans une autre langue est respectée.
- [x] 8.3 Lancer `openspec validate agent-language-settings --strict` puis la suite de tests (`go test ./...` dans `backend/`, `npm test` dans `frontend/`) et confirmer l'absence d'échec.
