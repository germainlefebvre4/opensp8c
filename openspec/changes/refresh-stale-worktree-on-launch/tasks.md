# Tasks

## 1. Contrôles git du worktree

- [ ] 1.1 Ajouter dans `internal/pool/worktree.go` une méthode `ChangeCommitted(change) (bool, error)` (`git cat-file -e HEAD:openspec/changes/<change>/tasks.md`, code de sortie interprété) ; vérifier par un test dans `worktree_test.go` : fichier committé → `true`, fichier seulement non suivi ou modifié non committé → `false`, erreur git hors code attendu → erreur
- [ ] 1.2 Ajouter `RecreateFromHead(change)` : retrait du worktree sans force, `git branch -d`, `worktree add -b` depuis HEAD, ré-enregistrement de la base ; vérifier par des tests : branche périmée sans travail → nouveau worktree contenant `tasks.md` et base ré-enregistrée ; branche avec commit propre ou worktree sale → erreur sans rien supprimer

## 2. Ordre et raisons dans le worker

- [ ] 2.1 Dans `runWorker`, avant `Provision` et quand `feature/<change>` n'existe pas, pauser avec « doit être committé » si `ChangeCommitted` est faux ; vérifier par un test de worker (`worker_test.go`) : change non committé → worker en pause, aucune branche ni worktree créés, aucun agent lancé
- [ ] 2.2 Après `Provision`, si `tasks.md` manque au worktree et que le change est committé dans HEAD : recréer via `RecreateFromHead` quand `HasWork` est faux puis rejouer la vérification une fois ; sinon pauser avec la nouvelle raison « la branche ne contient pas le change » ; vérifier par des tests de worker : branche périmée sans travail → agent lancé ; branche périmée avec travail → pause à raison distincte, branche et worktree intacts ; change committé après une première pause → reprise réussie
- [ ] 2.3 Conserver la raison « doit être committé » quand HEAD ne contient pas le change même avec une branche existante ; vérifier par un test de worker dédié

## 3. Documentation et vérification d'ensemble

- [ ] 3.1 Mettre à jour la section pool du `README.md` (message « doit être committé » et nouveau comportement de branche périmée) ; vérifier que le texte du README correspond aux deux raisons de pause du code
- [ ] 3.2 Vérifier le scénario de bout en bout : change créé non committé, lancement → pause sans orphelin, `git commit` du change, reprise → l'agent démarre ; `go test ./...` côté backend passe
