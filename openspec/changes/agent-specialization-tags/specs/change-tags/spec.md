# Spec Delta

## MODIFIED Requirements

### Requirement: Tags sémantiques stockés dans `.openspec.yaml`
Chaque change SHALL pouvoir porter une section `tags` dans son `.openspec.yaml` contenant quatre champs : `type` (chaîne parmi `frontend`, `backend`, `batch`, `fullstack`), `complexity` (entier 1–5), `components` (tableau de chaînes kebab-case) et `agent_specialization` (tableau de chaînes kebab-case). Les champs `_auto` (booléen) et `_tagged_at` (date ISO) SHALL également être présents pour tracer l'origine de la dérivation. La section `tags` est optionnelle — son absence est valide et ne produit aucune erreur.

#### Scenario: Change avec tags complets
- **WHEN** un `.openspec.yaml` contient une section `tags` avec `type`, `complexity`, `components` et `agent_specialization`
- **THEN** le backend parse ces valeurs et les expose dans la réponse API du change (champ `tags`)

#### Scenario: Change sans section tags
- **WHEN** un `.openspec.yaml` ne contient pas de section `tags`
- **THEN** le champ `tags` dans la réponse API est `null` ou absent, sans erreur

#### Scenario: Change tagué avant l'introduction du champ agent_specialization
- **WHEN** un `.openspec.yaml` contient une section `tags` avec `type`, `complexity` et `components` mais sans `agent_specialization` (changes tagués avant ce champ)
- **THEN** le backend parse la section normalement, sans erreur, et expose `agent_specialization` comme tableau vide dans la réponse API

## ADDED Requirements

### Requirement: Dérivation automatique de la spécialisation agent via LLM contrainte à un vocabulaire fermé
Le service de tagging SHALL dériver `agent_specialization` via le même appel LLM que celui utilisé pour `complexity` et `components` (CLI `claude --print` sur `proposal.md`/`design.md`), mais SHALL contraindre la réponse à ne choisir que parmi le vocabulaire fermé courant (liste de base de la capability `agent-specialization` + extensions utilisateur), sans jamais inventer de nouvelle valeur — à la différence de `components`, dont le vocabulaire reste ouvert.

#### Scenario: Dérivation réussie parmi le vocabulaire fermé
- **WHEN** le tagger est déclenché pour un change ayant `proposal.md`
- **THEN** le LLM reçoit le vocabulaire fermé courant en contexte et retourne une ou plusieurs valeurs qui en font partie, écrites dans `agent_specialization`

#### Scenario: Valeur hors vocabulaire renvoyée par le LLM
- **WHEN** la réponse du LLM contient une valeur absente du vocabulaire fermé courant (hallucination)
- **THEN** le service filtre cette valeur avant écriture dans `.openspec.yaml`, de sorte que `agent_specialization` ne contienne jamais de valeur hors du vocabulaire fermé

#### Scenario: CLI claude indisponible
- **WHEN** la CLI `claude` n'est pas installée ou inaccessible
- **THEN** la dérivation de `agent_specialization` est ignorée silencieusement, comme pour `complexity` et `components`, sans affecter le reste du fonctionnement
