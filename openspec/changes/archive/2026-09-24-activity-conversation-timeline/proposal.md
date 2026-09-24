# Proposal

## Why

L'onglet **Log** du `DetailPanel` ne retrace aujourd'hui que les messages bruts d'un run `ff` sélectionné manuellement dans une liste, avec un rendu texte simplifié qui ignore silencieusement les appels d'outils (`tool_use`/`tool_result`) présents dans les données brutes. Rien ne permet de voir, pour une Task donnée, la chronologie complète de ce qui s'y est passé : quels outils l'agent a utilisés et combien de temps ils ont pris, mais aussi les événements qui ne viennent pas de l'agent (changement de colonne Kanban, case `tasks.md` cochée, prise en charge par un worker du pool, commit git manuel). Un utilisateur qui veut comprendre "que s'est-il passé sur cette task" doit aujourd'hui recouper plusieurs sources disjointes (onglet Log, onglet Tasks, panneau du pool, `git log`), ou n'a tout simplement pas l'information (aucune des sources non-agent n'est journalisée avec un timestamp exploitable).

## What Changes

- Renommage de l'onglet **Log** du `DetailPanel` en **Conversation**, dont le contenu n'est plus limité à un run `ff` choisi manuellement mais fusionne, en un seul fil chronologique, tous les événements liés à une Task : messages/appels d'outils de l'agent (tous runs confondus : explore, ff, apply, verify, archive), transitions de statut Kanban, cases `tasks.md` cochées/décochées, changements de statut d'un worker du pool assigné à cette task, et commits git détectés sur la branche du change.
- Les blocs `tool_use`/`tool_result` déjà présents dans les JSONL de conversation (mais jusqu'ici ignorés) sont désormais parsés : chaque appel d'outil devient une entrée typée (nom de l'outil, ex. `Bash`/`Read`/`Edit`) avec une **durée calculée** (`ts(tool_result) - ts(tool_use)`, appariés par `tool_id`).
- Nouvelle **Activity Store** backend qui persiste, par change, les entrées non-agent qui ne sont aujourd'hui journalisées nulle part (transition Kanban, toggle de tâche, transition de statut worker, commit git détecté), afin de pouvoir les fusionner avec les entrées agent existantes.
- Nouvelle visualisation dans l'onglet Conversation : une frise chronologique monoligne où chaque segment est coloré selon le type d'action (un type = une couleur, légende associée) et dont la **largeur est proportionnelle à la durée réelle** de l'action, surmontant la liste détaillée des entrées (même palette de couleurs sur les badges de la liste).
- Mise à jour temps réel de l'onglet Conversation pendant qu'une Task est active (agent en cours, worker du pool assigné), en réutilisant le mécanisme SSE existant (`/api/workspaces/{id}/events`) plutôt qu'un polling ad-hoc.

Hors périmètre (explicitement exclu) : rétro-remplissage de l'historique des commits/toggles antérieurs à ce change (l'Activity Store démarre vide et se peuple à partir des nouveaux événements) ; toute action corrective ou de rejeu depuis l'onglet Conversation (lecture seule, comme l'onglet Log actuel) ; la vue "Agents" multi-workspace (`agent-pool-visibility`), qui reste une liste de statuts sans lien avec cette timeline par task.

## Capabilities

### New Capabilities
- `activity-timeline`: Activity Store backend par change (schéma d'entrée, persistance, endpoint de lecture fusionnée, détection de commits git manuels, événement SSE d'ajout) et onglet **Conversation** du DetailPanel (liste unifiée, frise monoligne colorée par type avec largeur proportionnelle à la durée, légende, filtre par type).

### Modified Capabilities
- `conversation-store`: la 4e exigence (onglet **Log**, sélection manuelle d'un run `ff`, rendu texte simplifié) est retirée — remplacée par l'onglet **Conversation** de `activity-timeline`. Les JSONL de conversation et leurs endpoints de listing/lecture par run restent inchangés et deviennent une source lue par `activity-timeline` ; les blocs `tool_use`/`tool_result` qu'ils contiennent déjà, jusqu'ici ignorés au parsing, sont désormais exploités.
- `workspace-events`: le stream SSE existant se voit ajouter un événement `activity_appended` (par change), émis à chaque nouvelle entrée dans l'Activity Store, pour que l'onglet Conversation se mette à jour en temps réel sans polling.
- `session-log-retention`: le job de purge périodique existant, qui supprime déjà `conversations/<workspaceId>/<changeName>/**` après archivage, est étendu pour purger de la même façon `activity/<workspaceId>/<changeName>/**` (nouveau répertoire de l'Activity Store), avec la même politique de délai (`changeLogRetentionDays`).

## Impact

- Backend : nouveau package `backend/internal/activity` (store, schéma d'entrée, parsing des `tool_use`/`tool_result` depuis `conversation.Store`, détection périodique de commits git sur la branche du change) ; nouveau(x) endpoint(s) de lecture fusionnée (probablement `backend/internal/api/handlers/activity.go`) ; points d'émission ajoutés dans `backend/internal/api/handlers/task.go` (`PatchTask`, toggle de tâche) et `backend/internal/api/handlers/ff.go` (`TriggerFF`, `ResetTasks`) — `kanban_status` étant dérivé de `tasks_done`/`tasks_total` plutôt que muté directement, c'est la mutation sous-jacente (toggle, déclenchement ff, reset) qui est journalisée, pas un "changement de statut" au sens strict ; et dans `backend/internal/pool/manager.go` (au même point que `broadcastLocked`, transitions de statut worker) ; nouvel événement SSE dans `backend/internal/watcher/watcher.go` ; extension du job de purge existant (`session-log-retention`) pour couvrir le nouveau répertoire `activity/`.
- Frontend : `frontend/src/components/DetailPanel.tsx` (renommage de l'onglet, remplacement du sélecteur de run par la vue unifiée) ; nouveau composant de frise monoligne + liste ; nouveau hook de lecture de la timeline fusionnée (remplace ou étend `useConversationRun.ts`/`useConversationRuns.ts`) ; mise à jour des clés i18n `frontend/src/locales/{en,fr}/detailPanel.json` (`tabs.log` → `tabs.conversation`, libellés des types d'action).
- Aucun changement de format des JSONL de conversation existants sur disque (lecture enrichie, écriture inchangée).
