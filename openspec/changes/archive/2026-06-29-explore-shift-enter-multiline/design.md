## Context

Les panels d'exploration (`ExplorePanel` et `ExploreAnonymousPanel`) utilisent un `<input type="text">` pour la saisie des messages. Ce composant HTML est intrinsèquement monoligne : Enter soumet le formulaire, aucun mécanisme natif ne permet d'insérer une nouvelle ligne.

L'exploration conversationnelle bénéficie de messages structurés et articulés. La contrainte monoligne pousse l'utilisateur à condenser sa pensée alors que l'interface vise justement à encourager une réflexion développée.

Les deux panels ont une structure identique (même form, même input, même handler `handleSend`), donc le changement s'applique symétriquement.

## Goals / Non-Goals

**Goals:**
- Permettre la saisie multiligne via `Shift+Enter`
- Conserver `Enter` seul pour l'envoi
- Auto-resize du textarea selon le contenu
- S'appliquer aux deux panels (ExplorePanel et ExploreAnonymousPanel)

**Non-Goals:**
- Modifier le backend ou le format des messages envoyés
- Ajouter du formatage riche (markdown input, bold, etc.)
- Persister le brouillon entre sessions

## Decisions

### Textarea à la place de l'input

Remplacer `<input type="text">` par `<textarea>`. C'est le seul composant HTML natif supportant la saisie multiligne. Alternatives écartées :

- **contenteditable div** : complexité accrue pour gérer la valeur React, le curseur, le submit — aucun bénéfice ici
- **librairie tierce** (Slate, ProseMirror) : overkill pour un textarea de chat basique

### Gestion de l'envoi via onKeyDown

Le formulaire `onSubmit` ne distingue pas Enter de Shift+Enter. Il faut intercepter l'événement au niveau du textarea :

```
onKeyDown:
  si Enter et pas Shift → preventDefault + submit
  si Shift+Enter → laisser le comportement natif (newline)
```

Le `onSubmit` du form reste en place pour le bouton "Envoyer".

### Auto-resize via useEffect sur ref

À chaque changement de `input`, réinitialiser `height: auto` puis assigner `scrollHeight` pour que le textarea s'agrandisse exactement à la hauteur du contenu. Pas de librairie externe nécessaire.

```
ref.current.style.height = 'auto'
ref.current.style.height = ref.current.scrollHeight + 'px'
```

Une hauteur max CSS (`max-height`) limite l'expansion et active le scroll interne au-delà.

## Risks / Trade-offs

- **Hauteur initiale** : un textarea vide peut paraître plus haut qu'un input. Mitigation : `rows="1"` + height pilotée par le JS auto-resize.
- **Reset après envoi** : après `send()`, `setInput('')` doit aussi déclencher le recalcul de hauteur. Mitigation : inclure la réinitialisation dans le `useEffect` qui dépend de `input`.
- **Accessibilité** : `<textarea>` est natif et accessible, pas de régression.
