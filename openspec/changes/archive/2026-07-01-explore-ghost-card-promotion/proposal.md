## Why

Le flux d'exploration anonyme promut les changes de manière autonome (décision IA) sans que l'utilisateur ait validé cette intention, et sans conserver la conversation pour une reprise ultérieure. C'est le 2e vecteur d'usage principal de l'app — si ce flux n'est pas maîtrisé par l'utilisateur, l'app se réduit à un simple Kanban.

## What Changes

- Le premier message envoyé dans une session anonyme crée un **ghost card** dans la colonne "to-explore", visible dans le kanban, sans créer de dossier workspace
- Le LLM nomme le ghost card via un event `ghost_named` en début de première réponse (nom kebab-case dérivé du contexte)
- La conversation est persistée côté frontend (localStorage) — aucun fichier dans le workspace pendant la phase d'exploration
- La session expirée peut être reprise : le contexte est réinjecté depuis localStorage (verbatim si < 60K chars, tronqué sinon)
- La promotion vers un change réel est **déclenchée par l'utilisateur** via un drag du ghost card vers la colonne "todo", avec dialog de confirmation
- La promotion lance FF dans la session existante (contexte intact) ou avec contexte injecté (session expirée) — le dossier openspec est créé uniquement à ce moment
- Un bouton delete (avec confirmation) est ajouté sur le ghost card et dans le panel d'exploration
- **BREAKING** : la promotion automatique (LLM émet `change_created` → change créé sans action utilisateur) est supprimée pour les sessions anonymes

## Capabilities

### New Capabilities

- `explore-ghost-card` : Entité ghost card créée au premier message d'une session anonyme — record dans preferences.json, carte visible en "to-explore" avec badge "exploring", aucun fichier workspace
- `exploration-conversation-persistence` : Persistance de la conversation d'exploration en localStorage (frontend uniquement, clé par ghost card ID), avec stratégie d'injection au resume (verbatim / tronquée)
- `exploration-promote-to-change` : Flux de promotion humain — drag ghost card → "todo" → dialog de confirmation → FF dans session existante ou avec contexte injecté → change réel créé dans "todo"

### Modified Capabilities

- `anonymous-explore-session` : La promotion automatique (change_created) est remplacée par la création du ghost card au premier message + event ghost_named pour le nommage. Le LLM ne crée plus de change de manière autonome.
- `kanban-drag-drop` : Les ghost cards (status "exploring") en "to-explore" sont draggables vers "todo" via le flux promote (différent du FF direct existant).
- `explore-bottom-panel` : Ajout d'un bouton delete avec dialog de confirmation pour les sessions d'exploration ghost.

## Impact

- `backend/internal/session/manager.go` : suppression du pattern change_created auto pour les sessions anonymes, ajout event ghost_named, création du ghost record au premier message
- `backend/internal/preferences/preferences.go` : ExplorationStore — CRUD des ghost records (id, wsId, name, sessionId, createdAt)
- `backend/internal/api/handlers/explore.go` : endpoint POST /promote pour envoyer /opsx:ff dans la session existante
- `backend/internal/api/handlers/ff.go` : TriggerFF sélectionne session existante si active, sinon démarre subprocess avec contexte injecté
- `backend/internal/api/router.go` : nouvelles routes explore (promote, delete ghost)
- `backend/internal/watcher/watcher.go` : events ghost_named, ghost_card_created, exploration_deleted
- `frontend/src/hooks/useAnonymousExploreSession.ts` : détection ghost_named, sauvegarde messages localStorage
- `frontend/src/hooks/useExploreSession.ts` : injection contexte localStorage au reconnect
- `frontend/src/components/ExploreAnonymousPanel.tsx` : trigger création ghost au 1er message
- `frontend/src/components/ExploreBottomPanel.tsx` : bouton delete + dialog confirmation
- `frontend/src/components/KanbanColumn.tsx` : bouton delete sur ghost cards
- `frontend/src/pages/KanbanPage.tsx` : handlePromote (drag ghost → todo), handleDeleteGhost, dialog confirmation
- `frontend/src/lib/api.ts` : endpoints promote, deleteGhost
- Dépendance npm : aucune
