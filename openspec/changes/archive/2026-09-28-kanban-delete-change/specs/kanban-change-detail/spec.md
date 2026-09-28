# Spec Delta

## ADDED Requirements

### Requirement: Afficher l'onglet Actions dans le DetailPanel
Le `DetailPanel` SHALL afficher un onglet **Actions**, à côté des onglets Tâches, Proposal, Design, Log et Tags. Cet onglet SHALL regrouper les actions de cycle de vie disponibles pour le change ouvert. Son contenu SHALL varier selon le `kanban_status` du change : le bouton Supprimer est toujours présent (sauf statut `archived`), le bouton Archiver n'apparaît qu'au statut `done`, et les boutons "Approuver & Fusionner" / "Demander correction" n'apparaissent qu'au statut `to-review`.

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
- **THEN** seul le bouton Supprimer est affiché

### Requirement: Supprimer un change depuis le DetailPanel
L'utilisateur SHALL pouvoir supprimer définitivement un change depuis le bouton Supprimer de l'onglet Actions, pour tout change dont le `kanban_status` est `todo`, `in-progress`, `to-review` ou `done`. La suppression n'est pas proposée pour un change en statut `archived`. La suppression SHALL être protégée par une confirmation explicite de l'utilisateur avant d'être déclenchée. Si un worker de l'Agent Pool Orchestrator est actif sur ce change, le bouton Supprimer SHALL être désactivé. Si le change possède un ghost record associé encore actif (exploration non "figée"), ce ghost record SHALL être supprimé avec le change.

#### Scenario: Clic sur Supprimer ouvre une confirmation
- **WHEN** l'utilisateur clique sur le bouton Supprimer dans l'onglet Actions
- **THEN** une boîte de dialogue de confirmation s'affiche, nommant le change à supprimer

#### Scenario: Confirmation déclenche la suppression
- **WHEN** l'utilisateur confirme la suppression
- **THEN** le change est supprimé, le DetailPanel se ferme, et la carte disparaît du Kanban

#### Scenario: Annulation de la confirmation
- **WHEN** l'utilisateur annule la boîte de dialogue de confirmation
- **THEN** rien n'est supprimé et le DetailPanel reste ouvert sur le change

#### Scenario: Bouton désactivé si un worker est actif
- **WHEN** un worker de l'Agent Pool Orchestrator est actif sur le change ouvert
- **THEN** le bouton Supprimer est désactivé

#### Scenario: Suppression cascade le ghost associé
- **WHEN** l'utilisateur confirme la suppression d'un change ayant un ghost record associé encore actif
- **THEN** le change et son ghost record associé sont tous deux supprimés, et la carte ghost disparaît également de la colonne To Explore

#### Scenario: Erreur de suppression
- **WHEN** la requête de suppression échoue
- **THEN** un message d'erreur est affiché dans l'onglet Actions et le change n'est pas supprimé

### Requirement: Endpoint de suppression d'un change
Le backend SHALL exposer un endpoint `DELETE /api/workspaces/{id}/changes/{name}` qui supprime définitivement le dossier `openspec/changes/{name}/`. Si un worker de l'Agent Pool Orchestrator est actif sur ce change, l'endpoint SHALL refuser la suppression. Si un ghost record du workspace porte le même nom que le change, l'endpoint SHALL également le supprimer (arrêt de session, retrait de `preferences.json`, suppression du brouillon et des logs associés).

#### Scenario: Suppression réussie
- **WHEN** une requête `DELETE /api/workspaces/{id}/changes/{name}` cible un change existant sans worker actif
- **THEN** le backend supprime le dossier du change et retourne HTTP 204

#### Scenario: Change inexistant
- **WHEN** la requête cible un change qui n'existe pas
- **THEN** le backend retourne HTTP 404

#### Scenario: Worker actif sur le change
- **WHEN** la requête cible un change sur lequel un worker de l'Agent Pool Orchestrator est actif
- **THEN** le backend retourne HTTP 409 et ne supprime rien

#### Scenario: Ghost record associé
- **WHEN** la requête cible un change pour lequel un ghost record du même workspace porte le même nom
- **THEN** le backend supprime aussi ce ghost record avant de retourner HTTP 204

## MODIFIED Requirements

### Requirement: Archiver un change depuis le DetailPanel
L'utilisateur SHALL pouvoir déclencher l'archivage d'un change en statut **Done** depuis le bouton Archiver de l'onglet Actions du DetailPanel.

#### Scenario: Archivage depuis l'onglet Actions
- **WHEN** l'utilisateur clique sur "Archiver" dans l'onglet Actions du DetailPanel d'un change en statut **Done**
- **THEN** le change est archivé et le DetailPanel se ferme

#### Scenario: Erreur d'archivage
- **WHEN** l'archivage échoue
- **THEN** le message d'erreur est affiché dans l'onglet Actions
