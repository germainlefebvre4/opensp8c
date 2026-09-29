# Design

## Context

Le handler `TriggerGenerate` passe `docsgen.BuildPrompt(...)` comme `extraSystemPrompt` à `session.StartSubprocess`, qui l'injecte dans les arguments de la CLI de l'agent (`--append-system-prompt`). L'agent est déjà lancé avec `cmd.Dir = workspacePath`, donc `openspec/specs/` est accessible en lecture. Voir proposal.md pour le symptôme (E2BIG, limite de 128 Ko par argument sous Linux).

## Goals / Non-Goals

**Goals:**
- Prompt de taille bornée, indépendante du nombre/volume de specs.
- Aucun changement d'API, de handler ni de contrat du run one-shot.

**Non-Goals:**
- Corriger d'autres usages d'`extraSystemPrompt` volumineux (explore/ff) : à évaluer séparément.
- Garantir que l'agent lit exhaustivement chaque spec (délégué à l'agent).

## Decisions

- **L'agent lit les specs lui-même.** `BuildPrompt` retourne uniquement `DocsFormalismPrompt`, dont le texte demande de lire tous les fichiers `openspec/specs/*/spec.md` du projet (relatifs au répertoire courant) avant de rédiger. Le message utilisateur initial reste inchangé.
  - *Alternative B — specs dans le message utilisateur (stdin)* : contourne la limite d'argument mais envoie ~80k tokens à chaque run et se comporte différemment selon les agents (gemini, antigravity) ; écartée.
  - *Alternative C — fichier temporaire concaténé* : ajoute un fichier à créer/nettoyer pour un résultat équivalent à A ; écartée.
- **`BuildPrompt` conserve sa signature** `(workspacePath string) (string, error)` pour limiter le diff côté handler ; le paramètre reste inutilisé ou sert à vérifier l'existence de `openspec/specs/`. Pas de vérification bloquante : un workspace sans spec doit démarrer (voir spec delta).

## Risks / Trade-offs

- [L'agent peut lire les specs partiellement ou dans le désordre] → le prompt exige explicitement de lister et lire tous les fichiers, et de trier par nom de capability.
- [Plus de tours d'outils, donc run plus long] → acceptable pour une génération à la demande.
- [Un agent sans outil de lecture de fichiers ne pourrait pas produire les pages] → tous les agents supportés écrivent déjà les pages sur disque, donc disposent d'outils fichier.
