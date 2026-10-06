# Tasks

## 1. Types et résolution

- [ ] 1.1 Créer `internal/verification` (`Override` à pointeurs `Conformity` / `UI`, `LaunchParams`, `Resolved`) et la fonction pure `Resolve(platform, workspace, change)` en cascade `change > workspace > plateforme > false`, étape par étape, avec les paramètres de lancement UI résolus champ par champ entre plateforme et workspace ; vérifier par un test table-driven (aucune valeur, Configuration seule, workspace qui désactive, change qui active, étapes mixtes, paramètres de lancement surchargés un à un, entrées `nil`) : `go test ./internal/verification`
- [ ] 1.2 Ajouter la validation (`uiBaseUrl` URL absolue `http` / `https`, valeurs vides équivalentes à l'absence, suppression des espaces autour de `uiStartCommand`) et vérifier par des tests couvrant `localhost:5173`, `ftp://hote`, `http://localhost:3000`, une chaîne d'espaces

## 2. Rôle `verifier`

- [ ] 2.1 Ajouter `RoleVerifier` à `preferences.Roles` (après `fixer`), à `Valid()` et à `Preset` (`sonnet` / `medium` pour Claude, aucun préréglage pour les autres agents) ; vérifier dans `roles_test.go` : le rôle est accepté, un rôle inconnu reste rejeté, la cascade global puis rôle le résout, le préréglage Claude s'applique, un `preferences.json` sans rôle `verifier` se charge sans erreur
- [ ] 2.2 Afficher le sixième rôle dans `RoleSettingsTable` (libellé de l'étape « Vérification ») pour Configuration > Colonnes et Settings > Colonnes, avec clés fr et en ; vérifier dans `RoleSettingsTable.test.tsx` (six lignes de rôle plus la ligne globale, préréglage `sonnet` / `medium` affiché) et par le test i18n existant (mêmes clés dans les deux locales)

## 3. Réglages de Configuration et de workspace

- [ ] 3.1 Ajouter `Preferences.VerificationDefaults` (booléens et paramètres de lancement `omitempty`) et `WorkspacePrefs.Verification` (pointeurs), étendre `isEmpty` et `clone`, et retirer les sections vides du fichier ; vérifier par des tests de persistance : fichier existant sans le champ, section retirée après réinitialisation de la dernière surcharge, workspace inconnu ignoré
- [ ] 3.2 Implémenter le patch partiel (`null` réinitialise, champ absent inchangé, valeur non booléenne rejetée) pour `SetVerificationDefaults` et `PatchWorkspace`, calqué sur `applyPoolPatch`, et l'inclure dans `ValidateGlobalUpdate` ; vérifier par des tests unitaires de patch (activation, réinitialisation par `null`, `"maybe"` rejeté sans modification, `uiBaseUrl` invalide rejeté)
- [ ] 3.3 Exposer `verificationDefaults` dans `GET /api/preferences` et l'accepter dans `PATCH /api/preferences`, et exposer `verification` dans `overrides`, `inherited` et `resolved` de `GET/PATCH /workspaces/{id}/settings` (`404` pour un workspace inconnu) ; vérifier dans `preferences_test.go` et un test de handler de réglages (surcharge enregistrée, héritage visible, réinitialisation, valeur invalide en `400`, un autre workspace non affecté)

## 4. Réglage de change

- [ ] 4.1 Ajouter `Verification *verification.Override` à `openspecMeta` et implémenter `openspec.SetVerification(changeRoot, patch)` qui retire le champ quand les deux valeurs sont retirées ; vérifier par des tests : écriture, retrait d'une valeur, retrait des deux sans `verification: {}` résiduel, `.openspec.yaml` absent initialisé avec le schéma `spec-driven`
- [ ] 4.2 Vérifier que `SetLaunched`, `ClearKanbanState`, `ReorderReady` et l'écriture du tagger préservent `verification` ; ajouter un test de non-régression par fonction d'écriture partant d'un `.openspec.yaml` qui contient un réglage de vérification
- [ ] 4.3 Ajouter `PATCH /workspaces/{id}/changes/{name}/verification` (corps `conformity` / `ui` à `true`, `false` ou `null`, champ inconnu rejeté en `400`, `404` pour un change ou un workspace inconnu, `409` pour un change archivé) avec la réponse `{ override, inherited, resolved }`, et déclarer la route dans `router.go` ; vérifier dans un test de handler (activation, retour à l'héritage, dernier réglage retiré, archivé en `409`, `uiStartCommand` en `400`, introuvable en `404`)
- [ ] 4.4 Ajouter `verification` (`override`, `inherited`, `resolved`) à `ChangeDetail` dans la réponse de `GET /workspaces/{id}/changes/{name}`, calculé par le handler à partir du `.openspec.yaml` du dépôt principal et des préférences ; vérifier dans un test de handler : sans réglage, avec réglage de change qui active alors que le workspace désactive, workspace qui surcharge la Configuration, réglage lu dans le dépôt principal même quand un worktree contient une ancienne copie
- [ ] 4.5 Mettre à jour `docs/opensp8c/architecture.md` (section configuration : les trois niveaux, l'ordre de résolution, l'emplacement de chaque niveau, le fait que le réglage de change se lit dans le dépôt principal) et vérifier que le texte correspond au comportement implémenté

## 5. Client frontend

- [ ] 5.1 Ajouter les types (`VerificationSettings`, `VerificationOverride`, `VerificationPatch`, `ChangeVerification`) et les champs `verificationDefaults` / `verification` à `lib/api.ts` (`Preferences`, `GlobalSettingsPatch`, `WorkspaceSettings`, `WorkspaceSettingsPatch`, détail de change), plus `patchChangeVerification(workspaceId, name, patch)` ; vérifier avec `npm run typecheck`
- [ ] 5.2 Ajouter un module `lib/verification.ts` (brouillon, patch minimal, validation de l'URL de base, état tri-état `inherit` / `on` / `off`) ; vérifier dans `lib/verification.test.ts` (patch ne contenant que les champs modifiés, `null` pour un retour à l'héritage, URL invalides refusées)
- [ ] 5.3 Créer le hook `useSetChangeVerification(workspaceId, name)` (mutation, message d'erreur du backend, invalidation de `['change-detail', id, name]`) et étendre `useChangeDetail` avec `verification` ; vérifier avec un test de hook (corps envoyé, `null` pour l'héritage, erreur exposée, invalidation au succès)

## 6. Écrans

- [ ] 6.1 Créer `TriStateSelect` (Hérité avec valeur héritée affichée / Activé / Désactivé) et `VerificationSettingsForm` (portées `global` et `workspace`, interrupteurs en portée `global`, tri-états en portée `workspace`, champs `uiStartCommand` et `uiBaseUrl`, mention du coût en tokens, avertissement si `ui` est résolue à `on` sans commande de lancement, erreurs de validation) ; vérifier dans `VerificationSettingsForm.test.tsx` (valeurs initiales désactivées, valeur héritée affichée, surcharge marquée, réinitialisation, URL invalide bloquée)
- [ ] 6.2 Ajouter le sous-onglet « Vérification » à `ConfigurationPage` ; vérifier dans `ConfigurationPage.test.tsx` (sous-onglet visible, enregistrement envoie `verificationDefaults`, erreur du backend affichée)
- [ ] 6.3 Ajouter le sous-onglet « Vérification » à `SettingsPage` (cinq sous-onglets, `tab=verification` dans l'URL, ordre Agent Pool, Colonnes, Vérification, Environnement, Spécialisations) ; vérifier dans `SettingsPage.test.tsx` (sous-onglet visible, `tab=verification` rouvre le sous-onglet, surcharge envoyée dans `verification`, réinitialisation envoie `null`)
- [ ] 6.4 Ajouter la section Vérification à l'onglet Actions de `DetailPanel` (deux `TriStateSelect`, enregistrement immédiat, erreur affichée sans changer la sélection, absente pour un change archivé) ; vérifier dans `DetailPanel.test.tsx` (valeur héritée affichée, désactivation envoie `ui: false`, retour à l'héritage envoie `null`, erreur conservant la sélection, section absente pour un archivé, boutons de cycle de vie inchangés)
- [ ] 6.5 Ajouter les clés fr et en (`configuration`, `settings`, `detailPanel`) pour les libellés, la mention du coût en tokens, l'avertissement de commande manquante et les choix tri-états ; vérifier que les deux locales ont les mêmes clés (tests i18n existants) et que `npm test`, le lint et `npm run typecheck` sont verts

## 7. Vérification d'ensemble

- [ ] 7.1 Lancer `go test ./...` côté backend et la suite frontend complète (tests, lint, typecheck) ; tout doit être vert
- [ ] 7.2 Parcours manuel : activer `conformity` dans Configuration, la désactiver pour un workspace dans Settings, la réactiver pour un change depuis le DetailPanel ; vérifier à chaque niveau la valeur héritée affichée, le contenu de `preferences.json` et du `.openspec.yaml` du change, puis lancer et retirer du lancement ce change et constater que son réglage de vérification est conservé
