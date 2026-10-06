# Spec Delta

## MODIFIED Requirements

### Requirement: Afficher l'onglet Actions dans le DetailPanel
Le `DetailPanel` SHALL afficher un onglet **Actions**, à côté des onglets Tâches, Proposal, Design, Log et Tags. Cet onglet SHALL regrouper les actions de cycle de vie disponibles pour le change ouvert. Son contenu SHALL varier selon le `kanban_status` du change : le bouton Supprimer est toujours présent (sauf statut `archived`), le bouton Archiver n'apparaît qu'au statut `done`, et les boutons "Approuver & Fusionner" / "Demander correction" n'apparaissent qu'au statut `to-review`. Pour tout change non archivé, l'onglet SHALL en outre afficher une section **Vérification** décrite par l'exigence « Réglage de la vérification dans le DetailPanel », qui n'est pas une action de cycle de vie.

#### Scenario: Onglet Actions toujours présent
- **WHEN** le DetailPanel s'ouvre pour un change, quel que soit son statut
- **THEN** l'onglet Actions est visible dans la barre d'onglets

#### Scenario: Contenu de l'onglet Actions au statut Done
- **WHEN** l'onglet Actions est actif pour un change en statut **Done**
- **THEN** les boutons Supprimer et Archiver sont tous deux affichés

#### Scenario: Contenu de l'onglet Actions au statut To Review
- **WHEN** l'onglet Actions est actif pour un change en statut **To Review**
- **THEN** les boutons Supprimer, "Approuver & Fusionner" et "Demander correction" sont affichés

#### Scenario: Contenu de l'onglet Actions aux autres statuts
- **WHEN** l'onglet Actions est actif pour un change en statut **To Do** ou **In Progress**
- **THEN** seul le bouton Supprimer est affiché parmi les boutons d'action de cycle de vie
- **THEN** la section Vérification est affichée en plus

#### Scenario: Change archivé
- **WHEN** l'onglet Actions est actif pour un change archivé
- **THEN** aucune action de cycle de vie ni section de réglage n'est modifiable

## ADDED Requirements

### Requirement: Réglage de la vérification dans le DetailPanel
La section Vérification de l'onglet Actions SHALL permettre de définir, pour le change ouvert, chacune des étapes `conformity` et `ui` avec les choix « Hérité », « Activé » et « Désactivé ». Lorsque « Hérité » est sélectionné, la section SHALL indiquer la valeur héritée (Configuration et workspace). Un choix SHALL être enregistré immédiatement via `PATCH /api/workspaces/{id}/changes/{name}/verification`, et une erreur SHALL être affichée sans modifier la sélection enregistrée. La section SHALL être absente pour un change archivé.

#### Scenario: Valeur héritée affichée
- **WHEN** le workspace active `ui` et que le change n'a pas de réglage
- **THEN** le choix « Hérité » est sélectionné pour `ui` et indique la valeur héritée « Activé »

#### Scenario: Désactivation pour ce change
- **WHEN** l'utilisateur choisit « Désactivé » pour `ui`
- **THEN** une requête de réglage est envoyée avec `ui` à `false` et la section affiche le choix enregistré

#### Scenario: Retour à l'héritage
- **WHEN** l'utilisateur choisit « Hérité » pour une étape qu'il avait désactivée
- **THEN** une requête de réglage est envoyée avec `null` pour cette étape et la valeur héritée est affichée

#### Scenario: Erreur d'enregistrement
- **WHEN** l'enregistrement échoue
- **THEN** un message d'erreur est affiché et le choix précédent est conservé

### Requirement: Le détail d'un change expose ses réglages de vérification
La réponse de `GET /api/workspaces/{id}/changes/{name}` SHALL inclure un objet `verification` contenant `override` (valeurs définies par le change, champs absents si héritées), `inherited` (valeurs effectives sans le réglage du change) et `resolved` (valeurs effectives pour le change), chacune avec les booléens `conformity` et `ui` lorsque applicable.

#### Scenario: Change sans réglage
- **WHEN** un change n'a pas de réglage de vérification et que la Configuration n'en définit pas
- **THEN** `override` est vide et `inherited` et `resolved` valent `false` pour les deux étapes

#### Scenario: Change avec réglage
- **WHEN** un change définit `ui` à `true` alors que le workspace ne l'active pas
- **THEN** `override.ui` vaut `true`, `inherited.ui` vaut `false` et `resolved.ui` vaut `true`
