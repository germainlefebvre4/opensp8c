## Why

L'historique des changements (actifs + archivés) ne porte aucune métadonnée sémantique : impossible de répondre à "quels changements frontend ont touché la barre de recherche ?" ou de visualiser l'évolution d'un composant dans le temps. L'ajout d'une couche de tags sémantiques auto-dérivés rend l'historique requêtable et ouvre la voie à une vue timeline.

## What Changes

- `.openspec.yaml` étendu avec une section `tags` (type, complexity, components)
- Service de dérivation automatique des tags : heuristique pour le type (chemins de fichiers dans tasks.md), LLM pour la complexité et les composants (lecture de proposal.md + design.md)
- Vocabulaire des composants émergent : extrait dynamiquement de tous les YAMLs du workspace, fourni en contexte au LLM pour normaliser les noms (pas de doublons sémantiques)
- Batch rétroactif : tagging chronologique des 26+ changements archivés au démarrage si non tagués
- Nouveau flag `_auto: true` dans le YAML pour distinguer tags dérivés vs édités manuellement
- ChangeCard : affichage type + complexité (compact, toujours visible)
- DetailPanel : affichage complet des tags (type, complexité, liste des composants)
- Barre de recherche : filtrage par tags en plus du nom
- Nouvelle vue `/timeline` : historique chronologique filtrable par tags, avec heatmap des composants

## Capabilities

### New Capabilities

- `change-tags` : système de tags sémantiques — modèle de données dans `.openspec.yaml`, dérivation automatique (heuristique + LLM), vocabulaire émergent normalisé, batch rétroactif, API backend exposant les tags
- `change-timeline` : vue chronologique des changements filtrables par tags, triés par date de création, avec heatmap des composants les plus modifiés

### Modified Capabilities

- `kanban-change-detail` : ajout de la section Tags dans le detail panel (type, complexité, composants)
- `kanban-change-search` : extension du filtrage pour inclure les tags (type, composants) en plus du nom
- `kanban-board` : ajout des badges type + complexité sur les ChangeCards

## Impact

- Backend : extension du parser `change.go` (nouveaux champs tags dans le struct `Change`), nouveau service de dérivation (heuristique + appel LLM), endpoint de déclenchement manuel du tagging, extraction du vocabulaire courant
- Frontend : nouveaux types TypeScript pour les tags, mise à jour des composants ChangeCard et DetailPanel, extension du hook de filtrage, nouvelle page Timeline
- `.openspec.yaml` : ajout de la section `tags` (rétrocompatible — champ optionnel)
- Dépendance LLM : le tagging des composants et de la complexité requiert un appel à Claude API (ou subprocess Claude Code existant)
