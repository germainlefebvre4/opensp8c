## ADDED Requirements

### Requirement: Marqueur de génération réussie
À la fin d'un run de génération réussi, le système SHALL enregistrer sous `docs/opensp8c/` un marqueur contenant l'heure de début et l'heure de fin du run, avant d'annoncer la fin de la génération. Un run est réussi lorsque l'agent se termine sans erreur ET qu'au moins une page générée est présente sur disque. Un run en échec, ou terminé sans aucune page, SHALL NOT créer ni modifier le marqueur. Le marqueur SHALL NOT apparaître dans la liste des pages générées.

#### Scenario: Run réussi
- **WHEN** un run de génération se termine sans erreur et qu'au moins une page est présente sous `docs/opensp8c/`
- **THEN** le marqueur est écrit avec l'heure de début et l'heure de fin du run, avant que la fin de génération soit annoncée

#### Scenario: Run en échec
- **WHEN** un run de génération échoue, y compris après avoir écrit certaines pages
- **THEN** le marqueur existant reste inchangé (ou absent) et aucune nouvelle date de génération n'est enregistrée

#### Scenario: Run terminé sans page
- **WHEN** l'agent se termine sans erreur mais qu'aucune page n'est présente sous `docs/opensp8c/`
- **THEN** aucun marqueur n'est écrit

#### Scenario: Marqueur absent de la liste des pages
- **WHEN** les pages générées sont consultées pour un workspace disposant d'un marqueur
- **THEN** le marqueur n'est pas retourné comme page

## MODIFIED Requirements

### Requirement: Consultation des pages générées et de leur fraîcheur
Le système SHALL exposer un moyen de lister les pages actuellement présentes sous `docs/opensp8c/` d'un workspace, d'en lire le contenu, de connaître la date de fin du dernier run de génération réussi, et de déterminer si la documentation générée est potentiellement obsolète. La date de référence de génération SHALL être l'heure de fin du marqueur ; la référence de fraîcheur SHALL être l'heure de début du marqueur. En l'absence de marqueur, l'une et l'autre SHALL se replier sur la date de modification la plus récente parmi les pages générées. La documentation est potentiellement obsolète lorsqu'au moins un fichier `openspec/specs/**/spec.md` a une date de modification postérieure à la référence de fraîcheur. Sans page générée, aucune date de génération n'est exposée et la documentation n'est pas signalée comme obsolète.

#### Scenario: Liste des pages existantes
- **WHEN** les pages générées sont consultées pour un workspace
- **THEN** seules les pages effectivement présentes sous `docs/opensp8c/` sont retournées

#### Scenario: Date de dernière génération avec marqueur
- **WHEN** les pages générées sont consultées pour un workspace disposant d'un marqueur
- **THEN** la date de dernière génération retournée est l'heure de fin du marqueur

#### Scenario: Date de dernière génération sans marqueur
- **WHEN** les pages générées sont consultées pour un workspace dont les pages existent mais sans marqueur
- **THEN** la date de dernière génération retournée est la date de modification la plus récente des pages

#### Scenario: Aucune page générée
- **WHEN** aucune page n'est présente sous `docs/opensp8c/`
- **THEN** aucune date de dernière génération n'est retournée et la documentation n'est pas signalée comme obsolète

#### Scenario: Spec plus récente que la documentation générée
- **WHEN** un fichier `spec.md` sous `openspec/specs/` a une date de modification postérieure à l'heure de début du marqueur
- **THEN** la documentation générée est signalée comme potentiellement obsolète

#### Scenario: Spec modifiée pendant le run
- **WHEN** un fichier `spec.md` a été modifié après le début du dernier run réussi mais avant sa fin
- **THEN** la documentation générée est signalée comme potentiellement obsolète

#### Scenario: Retouche manuelle d'une page
- **WHEN** une page est modifiée à la main après le dernier run réussi, sans modification des specs depuis le début de ce run
- **THEN** la documentation générée n'est pas signalée comme obsolète et la date de dernière génération reste celle du marqueur

#### Scenario: Documentation générée à jour
- **WHEN** tous les fichiers `spec.md` sous `openspec/specs/` sont antérieurs à l'heure de début du marqueur
- **THEN** la documentation générée n'est pas signalée comme obsolète

#### Scenario: Repli sans marqueur
- **WHEN** aucun marqueur n'existe et qu'un fichier `spec.md` est plus récent que toutes les pages sous `docs/opensp8c/`
- **THEN** la documentation générée est signalée comme potentiellement obsolète
