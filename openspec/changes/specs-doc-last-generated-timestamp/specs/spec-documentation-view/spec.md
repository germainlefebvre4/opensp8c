## ADDED Requirements

### Requirement: Date de dernière génération de la documentation
Le sous-onglet "Documentation" SHALL afficher, dans le panneau latéral sous le bouton de régénération, la date et l'heure du dernier run de génération réussi, sous la forme "Générée le <date>". La date SHALL être formatée selon la langue de l'interface, en format court pour la date et pour l'heure (par exemple `01/10/2026 16:42` en français). Elle SHALL se rafraîchir automatiquement à la fin d'un nouveau run réussi, et SHALL NOT être modifiée par un run en échec. Aucune date SHALL être affichée lorsqu'aucune page n'a été générée.

#### Scenario: Documentation générée
- **WHEN** l'utilisateur ouvre le sous-onglet "Documentation" d'un workspace dont la dernière génération réussie date du 01/10/2026 à 16:42
- **THEN** le panneau latéral affiche "Générée le" suivi de cette date et de cette heure, formatées selon la langue de l'interface

#### Scenario: Nouvelle génération réussie
- **WHEN** un run de génération se termine avec succès alors que le sous-onglet est ouvert
- **THEN** la date affichée est remplacée par celle de ce run sans rechargement manuel

#### Scenario: Génération en échec
- **WHEN** un run de génération échoue
- **THEN** la date affichée reste celle du dernier run réussi

#### Scenario: Aucune page générée
- **WHEN** le répertoire `docs/opensp8c/` est vide ou absent
- **THEN** aucune date de génération n'est affichée

## MODIFIED Requirements

### Requirement: Indicateur de fraîcheur de la documentation générée
Le sous-onglet "Documentation" SHALL afficher un badge "documentation potentiellement obsolète" lorsqu'au moins un fichier `spec.md` sous `openspec/specs/` a été modifié après le début du dernier run de génération réussi (ou, en l'absence de trace de run, après la plus récente des pages générées sous `docs/opensp8c/`).

#### Scenario: Spec modifiée après la dernière génération
- **WHEN** un fichier `spec.md` sous `openspec/specs/` a une date de modification postérieure au début du dernier run de génération réussi
- **THEN** le badge "documentation potentiellement obsolète" est affiché dans le sous-onglet "Documentation"

#### Scenario: Documentation à jour
- **WHEN** tous les fichiers `spec.md` sous `openspec/specs/` sont antérieurs au début du dernier run de génération réussi
- **THEN** aucun badge n'est affiché

#### Scenario: Aucune page générée
- **WHEN** le répertoire `docs/opensp8c/` est vide ou absent
- **THEN** aucun badge de fraîcheur n'est affiché
