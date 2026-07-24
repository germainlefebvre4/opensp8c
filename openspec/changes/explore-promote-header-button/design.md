## Context

Actuellement, la conversion d'une session d'exploration anonyme (ghost card) en un change réel ne peut être déclenchée que par le drag-and-drop de la carte sur le Kanban. Cette approche manque de découvrabilité et oblige l'utilisateur à naviguer hors du chat pour formaliser son idée. 

Nous voulons ajouter un bouton discret "Créer le change" dans le header de `ExploreAnonymousPanel` pour simplifier cette transition.

## Goals / Non-Goals

**Goals:**
- Ajouter un bouton de promotion dans le header de `ExploreAnonymousPanel`.
- Assurer un comportement responsive selon la largeur du panneau (masquer le texte sous 350px, n'afficher que l'icône `✨` avec un tooltip) en utilisant les requêtes de conteneur (container queries) natives de Tailwind CSS v4.
- Réutiliser intégralement la logique et le dialogue de confirmation de promotion déjà existants dans `KanbanPage.tsx` via une prop de callback (`onPromote`).
- Assurer le support bilingue (français/anglais).

**Non-Goals:**
- Ne pas dupliquer le code du dialogue de confirmation de promotion.
- Ne pas modifier le backend, l'API existante `/api/workspaces/{id}/explorations/{ghostId}/promote` étant déjà entièrement fonctionnelle.
- Ne pas afficher ce bouton dans le volet d'exploration nommé (`ExplorePanel`) car celui-ci est déjà associé à un change existant.

## Decisions

### Décision 1 : Prop de callback `onPromote` pour réutilisation de la dialog
Pour éviter toute duplication de code et centraliser la gestion d'état, nous passerons une prop `onPromote` depuis `KanbanPage.tsx` jusqu'à `ExploreAnonymousPanel`. 
Au clic sur le bouton, `ExploreAnonymousPanel` appelle `onPromote()`, ce qui active le dialogue de confirmation dans la page Kanban. Cette architecture garde le composant d'exploration indépendant de la gestion d'état globale du Kanban.

### Décision 2 : Container Queries Tailwind CSS v4 pour le responsive
Le panneau d'exploration étant redimensionnable horizontalement, sa largeur ne correspond pas à celle de la fenêtre. Nous allons utiliser les Container Queries natives de Tailwind v4 :
1. Déclarer la classe `@container` sur l'élément racine de `ExploreAnonymousPanel`.
2. Appliquer les classes conditionnelles sur le texte du bouton : `hidden @[350px]:inline` pour n'afficher le texte que lorsque le panneau fait au moins 350px de large.

### Décision 3 : Localisation et traductions
Nous ajouterons de nouvelles clés dans les fichiers de traduction `explore.json` :
- `createChange` : "Créer le change" / "Create change"
- `createChangeTooltip` : "Créer un change à partir de cette exploration" / "Create a change from this exploration"

## Risks / Trade-offs

- **Risque :** Le bouton est cliqué alors que l'exploration vient de commencer et n'a pas de contexte.
  - **Atténuation :** Le dialogue de confirmation permet à l'utilisateur d'annuler s'il a cliqué par erreur. De plus, le bouton s'affiche uniquement si `ghostId` et `ghostName` sont définis, assurant qu'une session active existe.
- **Risque :** Manque d'espace dans le header pour les petits écrans.
  - **Atténuation :** Le masquage automatique du texte du bouton garantit que seul un bouton d'icône compact (`✨`) est affiché, évitant tout chevauchement avec le titre ou les boutons système (fermer, agrandir).
