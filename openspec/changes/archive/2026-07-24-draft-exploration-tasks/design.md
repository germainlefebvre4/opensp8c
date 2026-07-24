## Context

Actuellement, les sessions d'exploration anonymes d'opensp8c génèrent des Ghost Cards dans la colonne *To Explore*. L'historique de la conversation est stocké dans le `localStorage` du navigateur et dans des fichiers de logs d'exploration du backend, mais il n'existe aucun moyen de matérialiser les tâches et la description déduites par l'IA ou structurées par l'utilisateur avant la promotion finale (FF). Cette phase transitoire de "brouillon" (draft) manque de visibilité intermédiaire sur le Kanban et dans le panneau de chat.

Ce document détaille l'architecture pour stocker ces brouillons de manière isolée côté serveur (sans polluer le dépôt de code de l'utilisateur), les endpoints d'API associés, et l'intégration UI (split-screen dans le panel de chat).

## Goals / Non-Goals

**Goals:**
- **Isolation stricte** : Persister les brouillons (tâches et descriptions) côté serveur d'opensp8c sans créer de fichiers dans le répertoire du workspace actif avant la promotion.
- **REST API** : Exposer des routes d'API CRUD simples pour manipuler ces brouillons.
- **Live Sync** : Capter les marqueurs de tâche émis par le LLM (`ghost_draft_updated`) pour mettre à jour le fichier JSON et diffuser l'événement via SSE.
- **Split-Screen UX** : Fournir une interface de chat divisée montrant à la fois la conversation et le brouillon dynamique éditable.
- **Relais lors du Promote** : Consommer le fichier de brouillon pour initialiser le fichier `tasks.md` final lors de la promotion (Fast-Forward).

**Non-Goals:**
- Écrire des fichiers `.md` ou `.json` dans le projet de l'utilisateur durant la phase d'exploration.
- Gérer l'historique des commits ou des modifications Git pour ces brouillons.

## Decisions

### 1. Stockage Isolé côté Backend (`drafts/`)
Le serveur stocke les brouillons dans un dossier `drafts/` situé au même niveau que `preferences.json` (qui gère déjà les Ghost Cards).
- **Justification** : Évite absolument de modifier l'arbre Git de l'utilisateur. Garantit la persistance côté serveur même si l'utilisateur change de navigateur ou nettoie ses cookies (contrairement au `localStorage` pur).
- **Structure** : `drafts/<ghostId>.json`.

### 2. Endpoints d'API dédiés
Ajout de routes dans le routeur Chi du backend :
- `GET /api/workspaces/{id}/explorations/{ghostId}/draft`
- `PUT /api/workspaces/{id}/explorations/{ghostId}/draft`
- `DELETE /api/workspaces/{id}/explorations/{ghostId}/draft`

### 3. Capture du marqueur LLM et SSE
Durant le streaming, le serveur surveille `stdout` pour le marqueur `ghost_draft_updated`. S'il est intercepté :
1. Écrit le contenu dans `drafts/<ghostId>.json`.
2. Diffuse l'événement via `watcher.Broadcast(workspaceID, Event{Type: "draft_updated", Name: ghostID})`.

### 4. Promotion fluide
Lors de l'appel à `POST /promote`, si un fichier `drafts/<ghostId>.json` existe :
1. Le backend lit les tâches du brouillon.
2. Il injecte ces tâches au subprocess FF (ou les applique directement après la création de la structure OpenSpec) afin de pré-remplir instantanément le fichier `tasks.md` réel.
3. Supprime proprement le fichier `drafts/<ghostId>.json` du disque.

## Risks / Trade-offs

- **[Risk] Orphelins de fichiers de brouillons** → Si un utilisateur supprime un workspace ou si des Ghost Cards sont abandonnées, des fichiers peuvent rester dans le dossier `drafts/`.
  - *Mitigation* :
    1. Hook de nettoyage dans `DeleteGhost` pour s'assurer que le fichier JSON du brouillon est supprimé lorsque le Ghost Card est retiré du Kanban.
    2. Extension du balayage de rétention automatique (`RunRetentionSweep` dans `retention.go`) pour nettoyer les fichiers de brouillons dont les Ghost Cards associées n'existent plus ou sont obsolètes.

- **[Trade-off] Multi-utilisateurs** → Si deux utilisateurs éditent le brouillon en même temps, le dernier appel `PUT` l'emporte.
  - *Mitigation* : Étant donné qu'une session d'exploration est typiquement mono-utilisateur à un instant T (définie par l'identifiant unique de la Ghost Card), ce cas est extrêmement rare et acceptable pour un outil de développement local.
