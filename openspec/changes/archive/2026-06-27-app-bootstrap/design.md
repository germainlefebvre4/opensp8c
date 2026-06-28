## Context

OpenSP8C est une application desktop/web légère qui sert d'interface visuelle au workflow OpenSpec. Elle lit et écrit dans les répertoires `openspec/` de projets locaux, et orchestre Claude Code CLI en subprocess pour les sessions d'exploration. Il n'y a pas de code existant — le design part de zéro.

Contraintes structurantes :
- La machine hôte dispose des binaires `claude` et `openspec` installés
- Les projets (workspaces) sont des répertoires locaux, potentiellement des dépôts Git
- Le backend Go tourne sur la même machine que les projets
- Le statut Kanban n'existe pas dans le format OpenSpec natif — il doit être introduit

## Goals / Non-Goals

**Goals:**
- Lire les changements OpenSpec de n'importe quel workspace local et les afficher en Kanban
- Permettre une session de chat avec Claude Code (`/opsx:explore`) depuis l'UI
- Archiver un changement Done sans interaction utilisateur
- Gérer plusieurs workspaces depuis une seule instance backend

**Non-Goals:**
- Modifier les artifacts OpenSpec (proposal, design, tasks) depuis l'UI — c'est le rôle des skills Claude Code
- Authentification / multi-utilisateur
- Synchronisation distante ou cloud
- Gestion des commits Git

## Decisions

### 1. Architecture backend : un seul processus pour tous les workspaces

**Décision** : Un seul binaire Go sert tous les workspaces. Les routes sont préfixées par `workspaceId`.

**Alternatives considérées** :
- *Un processus Go par workspace* : plus isolé mais complexité de cycle de vie (spawn/kill), gestion des ports, IPC. Overkill pour un outil local.

**Rationale** : Simplicité opérationnelle. L'utilisateur lance une seule commande. Le backend maintient un registre en mémoire des workspaces lus depuis `config.yaml`.

---

### 2. Subprocess Claude Code : process long-lived par session

**Décision** : Un subprocess `claude` long-lived par session d'exploration, avec streaming JSON bidirectionnel.

```
claude --print \
       --input-format stream-json \
       --output-format stream-json \
       --include-partial-messages \
       --append-system-prompt "Never use AskUserQuestion or interactive choice prompts. Communicate only through plain conversational text." \
       --cwd <workspace-root>
```

Le Go backend proxifie stdin/stdout via WebSocket.

**Alternatives considérées** :
- *Spawn par tour avec `--resume <session-id>`* : latence à chaque message (spawn + init), gestion de l'ID de session, plus fragile.

**Rationale** : Latence nulle entre les messages. Le process reste en vie pendant toute la conversation. Le backend ferme stdin pour terminer proprement.

---

### 3. Statut Kanban : champ `kanban_status` dans `.openspec.yaml`

**Décision** : Le Go backend lit et écrit un champ `kanban_status` dans le fichier `.openspec.yaml` de chaque change.

Valeurs : `to-explore` | `todo` | `in-progress` | `done`

Valeur par défaut (si champ absent) : `to-explore`

**Alternatives considérées** :
- *Fichier séparé `.kanban.json`* : isole la donnée de l'app mais ajoute un fichier non-standard dans le répertoire du change.
- *Statut dérivé des artifacts* : fragile (parsing markdown variable, tasks inexistantes en début de cycle).

**Rationale** : Co-location avec les métadonnées existantes du change. Un seul fichier à lire. Le format YAML permet d'ajouter le champ sans casser la lecture par `openspec` CLI.

---

### 4. Temps réel : WebSocket pour le chat, polling REST pour le Kanban

**Décision** :
- Chat explore : WebSocket (bidirectionnel requis — l'utilisateur envoie des messages)
- État du Kanban (liste des changes, statuts) : polling REST toutes les 5s ou refresh sur action

**Alternatives considérées** :
- *SSE pour le chat* : unidirectionnel, ne permet pas d'envoyer les messages utilisateur via le même canal.
- *WebSocket pour tout* : complexité inutile pour le Kanban qui n'a pas besoin de push temps réel strict.

---

### 5. Frontend : React Query + useState local

**Décision** : React Query pour le state serveur (workspaces, changes, specs), useState/useContext pour l'état UI local (workspace actif, panel ouvert).

**Alternatives considérées** :
- *Zustand* : utile pour des stores globaux complexes, overkill ici.
- *Redux Toolkit* : trop verbeux pour cette taille d'app.

**Rationale** : React Query gère le cache, le polling, et les mutations HTTP nativement. L'état local reste dans les composants.

## Risks / Trade-offs

- **Interface Claude CLI instable** → Le format `--input-format stream-json` est récent. Une mise à jour de Claude Code pourrait changer le protocole. Mitigation : version-pin du binaire Claude dans la doc d'installation.

- **Subprocess long-lived et mémoire** → Une session d'exploration qui dure longtemps accumule du contexte LLM. Mitigation : timeout de session configurable (défaut : 30 min d'inactivité), bouton "Nouvelle session" dans l'UI.

- **`.openspec.yaml` et évolution du format OpenSpec** → Si OpenSpec CLI commence à valider strictement le fichier, le champ `kanban_status` pourrait être rejeté. Mitigation : monitoring des releases OpenSpec, migration vers fichier séparé si nécessaire.

- **Accès concurrent aux fichiers** → Le backend Go et Claude Code subprocess écrivent tous deux dans le répertoire du change. Risque de corruption si les deux écrivent simultanément. Mitigation : le backend n'écrit que `.openspec.yaml` ; les artifacts (proposal, design, tasks) sont réservés aux skills Claude Code.
