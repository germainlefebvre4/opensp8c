# spec-documentation-view Specification

## Purpose

Permet à l'utilisateur de consulter, dans un sous-onglet dédié de la vue Specs, une documentation projet lisible et illustrée de diagrammes, générée à partir des spécifications brutes, sans se substituer à la liste brute existante.

## Requirements

### Requirement: Sous-onglets Spécifications et Documentation
La vue Specs d'un workspace SHALL afficher deux sous-onglets : "Spécifications" (la liste brute existante des répertoires de `openspec/specs/`) et "Documentation" (les pages générées). Le sous-onglet "Spécifications" SHALL être sélectionné par défaut à l'ouverture de la vue.

#### Scenario: Ouverture de la vue Specs
- **WHEN** l'utilisateur ouvre la vue Specs d'un workspace
- **THEN** les sous-onglets "Spécifications" et "Documentation" sont affichés, avec "Spécifications" actif par défaut

#### Scenario: Comportement inchangé du sous-onglet Spécifications
- **WHEN** l'utilisateur consulte le sous-onglet "Spécifications"
- **THEN** la liste des specs brutes et leur contenu se comportent exactement comme la vue Specs actuelle (liste, contenu Markdown, TOC, lien historique)

### Requirement: Affichage des pages de documentation générées
Le sous-onglet "Documentation" SHALL afficher la liste des pages existantes sous `docs/opensp8c/` du workspace, dans un ordre fixe (overview, architecture, domain-model, workflows), en n'affichant que les pages effectivement présentes sur disque.

#### Scenario: Pages générées présentes
- **WHEN** l'utilisateur ouvre le sous-onglet "Documentation" d'un workspace disposant de pages sous `docs/opensp8c/`
- **THEN** seules les pages présentes sont listées, dans l'ordre overview, architecture, domain-model, workflows

#### Scenario: Aucune page générée
- **WHEN** le répertoire `docs/opensp8c/` est vide ou absent
- **THEN** le sous-onglet "Documentation" affiche un état vide invitant à générer la documentation, avec le bouton "Générer" visible

### Requirement: Rendu Markdown avec diagrammes Mermaid
Le contenu d'une page de documentation sélectionnée SHALL être rendu en Markdown, et tout bloc de code de langage `mermaid` qu'il contient SHALL être rendu comme un diagramme visuel plutôt que comme un bloc de code brut.

#### Scenario: Page contenant un diagramme Mermaid valide
- **WHEN** l'utilisateur sélectionne une page de documentation contenant un bloc ```mermaid``` valide
- **THEN** le diagramme correspondant est rendu visuellement dans le panneau de contenu

#### Scenario: Bloc Mermaid invalide
- **WHEN** un bloc ```mermaid``` présent dans une page ne peut pas être interprété
- **THEN** la vue affiche le bloc de code brut en repli, sans interrompre l'affichage du reste de la page

### Requirement: Déclenchement de la génération via un bouton
L'utilisateur SHALL pouvoir déclencher la génération de la documentation depuis le sous-onglet "Documentation" via un bouton "Générer", qui lance un run unique produisant l'ensemble des pages. Pendant l'exécution, l'interface SHALL afficher un état "en cours" et désactiver un nouveau déclenchement concurrent pour le même workspace.

#### Scenario: Déclenchement de la génération
- **WHEN** l'utilisateur clique sur "Générer"
- **THEN** un run de génération démarre pour le workspace et le bouton passe dans un état "en cours"

#### Scenario: Génération déjà en cours
- **WHEN** l'utilisateur ouvre ou recharge le sous-onglet "Documentation" pendant qu'un run de génération est déjà en cours pour ce workspace
- **THEN** l'état "en cours" est affiché et le bouton "Générer" est désactivé, sans démarrer de second run

#### Scenario: Génération terminée
- **WHEN** un run de génération se termine avec succès
- **THEN** la liste des pages du sous-onglet "Documentation" se rafraîchit automatiquement avec le contenu nouvellement généré

### Requirement: Indicateur de fraîcheur de la documentation générée
Le sous-onglet "Documentation" SHALL afficher un badge "documentation potentiellement obsolète" lorsqu'au moins un fichier `spec.md` sous `openspec/specs/` a été modifié plus récemment que la plus récente des pages générées sous `docs/opensp8c/`.

#### Scenario: Spec modifiée après la dernière génération
- **WHEN** un fichier `spec.md` sous `openspec/specs/` a une date de modification postérieure à toutes les pages sous `docs/opensp8c/`
- **THEN** le badge "documentation potentiellement obsolète" est affiché dans le sous-onglet "Documentation"

#### Scenario: Documentation à jour
- **WHEN** toutes les pages sous `docs/opensp8c/` sont plus récentes que tous les fichiers `spec.md` sous `openspec/specs/`
- **THEN** aucun badge n'est affiché

#### Scenario: Aucune page générée
- **WHEN** le répertoire `docs/opensp8c/` est vide ou absent
- **THEN** aucun badge de fraîcheur n'est affiché
