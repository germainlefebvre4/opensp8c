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
La TimelinePage en mode Matrice SHALL afficher, dans son panel droit, chaque spec sélectionnée avec sa timeline complète de changes inline, triée du plus récent au plus ancien. Ce requirement s'applique désormais dans le contexte de la **TimelinePage** et non de la SpecsPage.

#### Scenario: Spec avec plusieurs changes
- **WHEN** le panel droit de la Timeline est ouvert sur une spec ayant N changes liés
- **THEN** N lignes de timeline sont affichées, chacune indiquant le nom lisible du change (sans préfixe date), la date, et le statut

#### Scenario: Change actif dans la timeline
- **WHEN** un change dans la timeline a le statut "active"
- **THEN** il est affiché avec un indicateur visuel distinct le différenciant des changes archivés

#### Scenario: Spec sans aucun change dans le panel droit
- **WHEN** la spec sélectionnée dans la grille n'a aucun change lié
- **THEN** le panel droit indique qu'aucun change n'a touché cette spec

#### Scenario: Section orphelins
- **WHEN** la réponse de l'endpoint contient un tableau `orphans` non vide
- **THEN** une section dédiée affiche les noms des specs orphelines en bas de la grille (mode Matrice), pas dans le panel droit

### Requirement: Ouvrir le détail d'un change depuis la timeline
L'utilisateur SHALL pouvoir cliquer sur n'importe quel change dans la timeline spec pour ouvrir le DetailPanel de ce change dans le slot droit. Ce slot est situé dans la **TimelinePage** (mode Matrice) et non dans la SpecsPage.

#### Scenario: Clic sur un change archivé dans la timeline (mode Matrice)
- **WHEN** l'utilisateur clique sur un change archivé dans la timeline d'une spec en mode Matrice
- **THEN** le DetailPanel s'ouvre dans le slot droit de la TimelinePage avec les informations du change (proposal, design, tasks)

#### Scenario: Clic sur un change actif dans la timeline (mode Matrice)
- **WHEN** l'utilisateur clique sur un change actif dans la timeline d'une spec en mode Matrice
- **THEN** le DetailPanel s'ouvre dans le slot droit de la TimelinePage avec les informations du change actif

#### Scenario: Fermeture du DetailPanel depuis le mode Matrice
- **WHEN** l'utilisateur ferme le DetailPanel depuis le mode Matrice de la Timeline
- **THEN** le slot droit revient au panel de spec (timeline de la spec sélectionnée)
