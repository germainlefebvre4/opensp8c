# Tasks

## 1. Lecture git de la revue (backend)

- [x] 1.1 Ajouter à `WorktreeController` (`pool/worktree.go`) la résolution de la base de revue (`BaseBranch`, repli `CurrentBranch`, repli `HEAD` si la référence manque) et `ReviewFiles(change)` (`git diff --no-renames --numstat -z` + `--name-status -z` sur `<base>...feature/<change>`, fusion par chemin, statuts `added`/`modified`/`deleted`, `binary`) ; vérifier par `worktree_test.go` sur un dépôt temporaire : ajout, modification, suppression, binaire, nom avec espaces, renommage (suppression + ajout), base ayant avancé (fichiers de `main` absents de la liste), worktree supprimé
- [x] 1.2 Ajouter `ReviewPatch(change, path)` et `ReviewFile(change, path)` (patch unifié et `git show feature/<change>:<path>`) avec limite de 512 Kio (taille connue avant lecture pour le contenu), indicateurs `truncated` et `binary`, et validation du chemin par appartenance à `ReviewFiles` (`--` systématique) ; vérifier par des tests : fichier modifié, binaire, patch tronqué, chemin hors revue, `..`, chemin absolu, nom commençant par `-`, fichier supprimé (`ReviewFile` introuvable)

## 2. Endpoints de revue

- [x] 2.1 Créer le handler de revue et les routes `GET /workspaces/{id}/changes/{name}/review`, `.../review/diff?path=` et `.../review/file?path=` dans `router.go`, avec `404` (change inconnu), `409` `not_in_review` (statut ≠ `to-review`), `400` (chemin invalide), `404` (fichier hors revue), et `target_ahead` dans la liste ; vérifier par des tests de handler pour chaque code, le succès de chaque endpoint, et le fonctionnement pool arrêté et worktree absent
- [x] 2.2 Vérifier que les endpoints ne modifient ni la branche, ni le worktree, ni le marqueur de revue (comparer `git rev-parse` des branches et `git status` du worktree avant/après les appels) ; vérifier par un test dédié

## 3. Couche de données frontend

- [x] 3.1 Ajouter les types et les hooks `useChangeReview`, `useReviewDiff` et `useReviewFile` (clés de requête imbriquées sous celle du détail du change, chargement du diff et du contenu à la demande, erreur normalisée avec son code) ; vérifier par des tests de hook (succès, `409 not_in_review`, erreur réseau, invalidation par `change_updated`)

## 4. Rendu du diff

- [x] 4.1 Créer un composant `DiffView` qui parse le patch unifié avec `parsePatch` (lib `diff`) et rend hunks, numéros de lignes, lignes ajoutées/supprimées (style de `SpecEditor`), message pour binaire et pour patch tronqué ; extraire le rendu de ligne partagé dans `SpecEditor` seulement si cela ne change pas son comportement ; vérifier par `DiffView.test.tsx` (ajouts, suppressions, plusieurs hunks, binaire, tronqué) et `SpecEditor` inchangé si touché

## 5. Onglet Revue

- [x] 5.1 Créer `ReviewTab` : en-tête (branche, base, avertissement `target_ahead`), groupes OpenSpec/Code (groupe vide masqué), lignes de fichiers (chemin, statut, `+`/`-`), dépliage à la demande du diff, états de chargement, d'erreur avec réessai et de revue vide ; vérifier par `ReviewTab.test.tsx` pour chacun de ces cas
- [x] 5.2 Ajouter la bascule diff/rendu pour les fichiers `.md` non supprimés (vue diff par défaut, rendu via `Markdown`, chargement du contenu à la demande, message en cas d'échec ou de contenu tronqué, aucune bascule pour les autres fichiers) ; vérifier par des tests de `ReviewTab`
- [x] 5.3 Afficher le dernier run du change (date, issue) avec un lien vers la vue Agents réutilisant le paramètre de sélection de run exporté depuis `AgentsPage` (ou un module partagé), sans lien s'il n'y a pas de run ; vérifier par des tests (run présent : lien vers `/agents?...` correct ; aucun run : pas de lien)
- [x] 5.4 Ajouter les clés i18n fr/en de l'onglet (libellés, statuts, avertissements, états vides et d'erreur) ; vérifier que le test de parité des locales existant (ou un test ajouté) ne signale aucune clé manquante

## 6. Intégration au DetailPanel

- [x] 6.1 Ajouter l'onglet `review` à `DetailPanel.tsx`, visible uniquement pour `kanban_status === 'to-review'`, avec retour à l'onglet Tâches si le change quitte ce statut pendant que l'onglet est actif ; vérifier par `DetailPanel.test.tsx` (onglet présent en `to-review`, absent en `todo`, repli sur Tâches au changement de statut)

## 7. Vérification d'ensemble

- [x] 7.1 Vérifier de bout en bout sur un change réellement en revue : liste des fichiers cohérente avec `git diff --stat`, diff d'un fichier de code et d'un `.md` (bascule rendu), changement de branche cible avancée signalé, lien vers le run ouvrant son détail ; exécuter `go test ./...`, `npm test` et `openspec validate review-panel --strict` sans échec
