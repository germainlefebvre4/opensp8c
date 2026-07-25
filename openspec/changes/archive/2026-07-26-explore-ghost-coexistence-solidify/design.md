## Context

Le flux d'exploration actuel (`exploration-promote-to-change`) ferme la session d'exploration dès qu'un change est créé. Cela empêche les allers-retours itératifs de co-conception entre l'utilisateur et l'IA. Si les tâches générées ne conviennent pas, l'utilisateur doit supprimer le change, ré-ouvrir le chat, ré-expliquer et ré-générer. 

Pour fluidifier l'expérience, nous concevons un état intermédiaire de coexistence où l'exploration reste active (dans "À explorer") tandis que le change de travail est généré sous forme de "brouillon" (dans "À faire"). Une action humaine (Figer/Solidifier) ou physique (Commencer le code / Coche d'une tâche) résout et ferme définitivement l'exploration pour figer le change.

## Goals / Non-Goals

**Goals:**
- Conserver la session d'exploration active après la promotion par Fast-Forward.
- Afficher les changes créés sous un style de "brouillon" visuel (dashed, transparent) tant que leur exploration est active.
- Fournir un bouton "Figer" explicite sur la carte brouillon et dans le volet de détail.
- Assurer la solidification automatique lors du passage à "In Progress" ou du cochage d'une première tâche.
- Copier l'historique de chat de l'exploration au moment de la promotion pour que le change conserve sa documentation.

**Non-Goals:**
- Créer un nouvel état ou champ persistant de base de données dans OpenSpec. Tout l'état est dérivé dynamiquement de la coexistence des fichiers sur le disque et des préférences.
- Permettre de recréer un ghost à partir d'un change déjà solidifié (le chemin inverse est hors-scope).

## Decisions

### Décision 1 : Copie des logs d'exploration lors de la promotion
Actuellement, `MoveExplorationLogs` déplace physiquement le répertoire de logs. Pour maintenir la session d'exploration active, le backend doit **copier** les logs vers le dossier du change lors du FF, tout en laissant l'original intact pour les messages suivants dans l'exploration.
- **Détail :** Ajouter une méthode `CopyExplorationLogs` dans `backend/internal/conversation/store.go` et l'appeler dans `runPromoteFF`.

### Décision 2 : Préservation temporaire du Ghost et du Brouillon
Dans `runPromoteFF`, ne plus supprimer l'exploration de `preferences.json` ni supprimer le brouillon `.json`.
- **Ancien comportement :** Suppression à la fin de `runPromoteFF`.
- **Nouveau comportement :** Ne rien supprimer, émettre uniquement `ff_done` via SSE.

### Décision 3 : Dérivation dynamique du statut "Brouillon" (Draft)
Au lieu de modifier le schéma `.openspec.yaml` pour stocker un statut `draft`, le frontend dérive l'état "Brouillon" d'un change en vérifiant si un ghost card actif porte le même nom :
```typescript
const associatedGhost = changes.find(c => c.is_ghost && c.name === change.name)
const isDraft = !!associatedGhost
```
- **Rendu visuel :** Si `isDraft` est vrai, la carte dans la colonne "À faire" est affichée avec :
  - Une opacité de `opacity-80`.
  - Une bordure `border-dashed border-slate-300` (au lieu de blanche standard).
  - Un badge gris/violet compact "brouillon".
  - Un bouton discret `[Figer]` (ancre ou icône cadenas) qui appelle `deleteGhost(workspaceId, ghostId)`.

### Décision 4 : Solidification implicite
Pour éviter qu'un change ne reste éternellement un brouillon orphelin, la solidification s'exécute automatiquement lors de :
1. **Drag to In Progress :** Si un change brouillon est glissé vers une colonne autre que `todo`/`to-explore`, le frontend appelle silencieusement l'API `deleteGhost` pour figer le change avant d'effectuer l'action.
2. **Coche de tâche :** Si l'utilisateur coche une tâche dans `DetailPanel` sur un change qui est un brouillon, le frontend appelle silencieusement `deleteGhost` pour figer le change avant de toggler la tâche.

## Risks / Trade-offs

- **[Confusions de nom]** → Si l'utilisateur renomme l'exploration avant de la figer.
  - *Atténuation :* Le renommage de l'exploration est synchronisé. Le matching par nom reste robuste.
- **[Logs dupliqués]** → Les logs de l'exploration sont copiés vers le change au moment de la promotion. Si l'utilisateur continue d'explorer et de discuter, ces nouveaux messages ne seront pas automatiquement synchronisés dans le change d'implémentation.
  - *Atténuation :* C'est un comportement attendu : le change capture une "photo" de l'exploration au moment de la promotion. S'ils veulent intégrer de nouvelles consignes, ils peuvent figer le change courant et en créer un nouveau, ou relancer une promotion qui écrasera/mettra à jour le change.
