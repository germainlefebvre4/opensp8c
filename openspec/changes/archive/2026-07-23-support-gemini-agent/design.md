## Context

L'application `opensp8c` orchestre des agents de code sous forme de sous-processus. 
Actuellement, le backend d'exploration et de fast-forward est optimisé pour communiquer avec **Claude Code** en s'appuyant sur son protocole propriétaire d'échange de messages bidirectionnels structurés en JSON (`--input-format stream-json --output-format stream-json`).

L'agent **Gemini CLI** est déclaré comme supporté dans le backend, mais l'intégration est incomplète et dysfonctionnelle :
1. Le backend transmet des arguments spécifiques à Claude Code au binaire `gemini`, ce qui provoque son arrêt immédiat (erreur `Unknown arguments`).
2. Gemini CLI ne supporte pas le protocole de messagerie bidirectionnelle personnalisé de Claude Code sur son entrée standard (`--input-format stream-json`). À la place, il supporte le protocole standardisé **ACP (Agent Client Protocol)** via le drapeau `--acp` (basé sur JSON-RPC 2.0), ou une sortie brute unidirectionnelle en NDJSON (`--output-format stream-json`).

## Goals / Non-Goals

**Goals :**
- Permettre à l'utilisateur de sélectionner l'agent de code "Gemini" et d'ouvrir une session d'exploration fonctionnelle.
- Configurer correctement les arguments de ligne de commande passés à l'agent Gemini lors de l'initialisation du sous-processus.
- Assurer la transmission et la propagation des variables d'environnement (comme `GOOGLE_CLOUD_PROJECT`) aux processus agents enfants.

**Non-Goals :**
- Réimplémenter un interpréteur JSON-RPC 2.0 complet dans le backend d'exploration pour cette étape si une méthode de communication plus simple (par exemple via la traduction des messages ou l'utilisation d'un traducteur de protocole) est envisageable, ou si on se concentre d'abord sur la compatibilité de base de l'exécution.
- Modifier l'interface utilisateur (UI) du sélecteur d'agent ou ajouter des champs de saisie pour les variables d'environnement.

## Decisions

### Décision 1 : Spécialisation des arguments de démarrage (BuildSubprocessArgs)
Nous allons modifier la structure `AgentConfig` dans `backend/internal/agents/agents.go` pour permettre à chaque agent de définir ses propres arguments de démarrage via une méthode spécialisée ou un switch.

- **Pour Claude Code (`claude`) :**
  Garder les arguments d'origine :
  ```go
  []string{
      "--print",
      "--verbose",
      "--input-format", "stream-json",
      "--output-format", "stream-json",
      "--include-partial-messages",
      "--append-system-prompt", basePrompt,
  }
  ```
- **Pour Gemini CLI (`gemini`) :**
  Utiliser les options officiellement supportées :
  ```go
  []string{
      "--output-format", "stream-json",
      "--approval-mode", "auto_edit", // Évite les blocages interactifs
      "--skip-trust",                 // Fait confiance au workspace
  }
  ```
  *Note :* La gestion de `--session-id` et `--resume` dans `StartSubprocess` devra également s'assurer de ne pas casser le CLI Gemini.

### Décision 2 : Traducteur de flux de messages (Protocol Adapter)
Étant donné que le frontend et le gestionnaire de session d'`opensp8c` s'attendent à des événements JSON au format Claude, nous allons concevoir une couche d'adaptation au niveau du gestionnaire de flux (`backend/internal/session/manager.go` et `subprocess.go`) :
- **Entrée (User Message) :** Convertir le message JSON entrant `{ type: "user", message: { content: "..." } }` en texte brut ou format approprié pour l'entrée standard de Gemini.
- **Sortie (Agent Output) :** Traduire les lignes NDJSON émises par Gemini CLI (qui émet des événements de type `message` pour les deltas de texte) vers le format attendu par le frontend d'`opensp8c` (`content_block_delta` ou texte brut).

### Décision 3 : Propagation sécurisée de l'environnement
Nous allons explicitement documenter ou s'assurer que le backend Go n'écrase pas `cmd.Env` lors du démarrage de `exec.CommandContext`, garantissant ainsi que `GOOGLE_CLOUD_PROJECT` (et toute autre variable GCP requise par Gemini) soit transmise de manière transparente.

## Risks / Trade-offs

- **[Risque] Divergence des formats de streaming** → Si les événements émis par Gemini CLI changent de structure, la traduction pourrait casser.
  - *Atténuation :* Écrire un parseur résilient qui accepte aussi bien le texte brut que le JSON structuré, en s'appuyant sur les types génériques d'événements de Gemini CLI (`message`, `error`, `result`).
