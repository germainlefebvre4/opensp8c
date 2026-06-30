### Requirement: Endpoint d'aperçu des specs avec historique de changes
Le système SHALL exposer un endpoint `GET /api/workspaces/{id}/specs/overview` retournant pour chaque spec dans `openspec/specs/` la liste ordonnée des changes qui l'ont référencée (actifs et archivés), ainsi qu'une liste des specs orphelines (référencées dans des changes mais absentes de `openspec/specs/`).

#### Scenario: Workspace avec des changes archivés et actifs
- **WHEN** l'endpoint est appelé sur un workspace contenant des specs et des changes
- **THEN** la réponse contient un tableau `specs`, chaque entrée ayant un `name` et un tableau `changes` trié du plus récent au plus ancien

#### Scenario: Change actif touchant une spec
- **WHEN** un change actif (hors `archive/`) contient une spec dans son dossier `specs/`
- **THEN** le `ChangeRef` correspondant a `status: "active"`

#### Scenario: Change archivé touchant une spec
- **WHEN** un change dans `changes/archive/` contient une spec dans son dossier `specs/`
- **THEN** le `ChangeRef` correspondant a `status: "archived"`

#### Scenario: Spec sans aucun change lié
- **WHEN** une spec dans `openspec/specs/` n'est référencée dans aucun change
- **THEN** elle apparaît dans le tableau `specs` avec un tableau `changes` vide

#### Scenario: Spec orpheline
- **WHEN** un change référence une spec dans son dossier `specs/` mais cette spec est absente de `openspec/specs/`
- **THEN** son nom apparaît dans le tableau `orphans` de la réponse

#### Scenario: Workspace sans specs ni changes
- **WHEN** `openspec/specs/` est vide ou absent
- **THEN** la réponse retourne `{ "specs": [], "orphans": [] }`

### Requirement: Afficher la timeline des changes par spec en mode Historique
En mode Historique, la vue Specs SHALL afficher chaque spec avec sa timeline complète de changes inline, triée du plus récent au plus ancien.

#### Scenario: Spec avec plusieurs changes
- **WHEN** la vue est en mode Historique et une spec a N changes liés
- **THEN** N lignes de timeline sont affichées sous le nom de la spec, chacune indiquant le nom lisible du change (sans préfixe date), la date, et le statut

#### Scenario: Change actif dans la timeline
- **WHEN** un change dans la timeline a le statut "active"
- **THEN** il est affiché avec un indicateur visuel distinct le différenciant des changes archivés

#### Scenario: Spec sans aucun change en mode Historique
- **WHEN** la vue est en mode Historique et une spec n'a aucun change lié
- **THEN** la spec est mise en évidence visuellement (indicateur d'avertissement ou style distinct)

#### Scenario: Section orphelins
- **WHEN** la réponse de l'endpoint contient un tableau `orphans` non vide
- **THEN** une section dédiée affiche les noms des specs orphelines en bas de la vue Historique

### Requirement: Ouvrir le détail d'un change depuis la timeline
L'utilisateur SHALL pouvoir cliquer sur n'importe quel change dans la timeline pour ouvrir le DetailPanel de ce change dans le slot droit de la SpecsPage.

#### Scenario: Clic sur un change archivé dans la timeline
- **WHEN** l'utilisateur clique sur un change archivé dans la timeline
- **THEN** le DetailPanel s'ouvre dans le slot droit avec les informations du change (proposal, design, tasks)

#### Scenario: Clic sur un change actif dans la timeline
- **WHEN** l'utilisateur clique sur un change actif dans la timeline
- **THEN** le DetailPanel s'ouvre dans le slot droit avec les informations du change actif

#### Scenario: Fermeture du DetailPanel
- **WHEN** l'utilisateur ferme le DetailPanel depuis la vue Historique
- **THEN** le DetailPanel se ferme et la timeline reste affichée sans changement
