## ADDED Requirements

### Requirement: `tags.type` toujours exposé comme tableau dans l'API
Lorsqu'un change porte une section `tags`, le backend SHALL toujours exposer `tags.type` comme un tableau dans la réponse API, jamais comme `null` ni comme champ absent. Une section `tags` dont `type` est absent ou vide dans le `.openspec.yaml` SHALL produire `tags.type: []`.

#### Scenario: Tags sans clé `type`
- **WHEN** un `.openspec.yaml` contient une section `tags` (par exemple avec `complexity` et `components`) mais aucune clé `type`
- **THEN** le backend parse la section sans erreur et expose `tags.type` comme un tableau vide

#### Scenario: Tags avec `type` vide ou nul
- **WHEN** un `.openspec.yaml` contient `tags.type:` sans valeur, ou `tags.type: []`
- **THEN** le backend expose `tags.type` comme un tableau vide, jamais comme `null`

#### Scenario: Tags avec `type` renseigné
- **WHEN** un `.openspec.yaml` contient `tags.type: [frontend, backend]`
- **THEN** le backend expose `tags.type` comme un tableau contenant ces valeurs, inchangé
