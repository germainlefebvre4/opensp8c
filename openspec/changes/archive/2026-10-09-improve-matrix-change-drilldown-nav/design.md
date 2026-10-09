# Design

## Context

Dans `TimelinePage.tsx` (mode Matrice), le slot droit de 400px rend soit `DetailPanel` (si `selectedChange`), soit l'en-tête de spec + `SpecHistoryView` (si `selectedSpec`). `DetailPanel` est partagé avec le Kanban et n'expose qu'un callback `onClose`, appelé à la fois par le bouton X et après archivage, suppression ou demande de correction (`DetailPanel.tsx`). Dans la Matrice, `onClose` est branché sur « désélectionner le change », d'où l'ambiguïté du X. `SpecHistoryView` est démonté pendant le drill-down et accepte déjà `selectedChangeName` (surlignage), aujourd'hui toujours `null` quand la liste est visible. Voir proposal.md pour la motivation.

## Goals / Non-Goals

**Goals:**
- Deux gestes distincts dans le drill-down : « ← spec » remonte d'un niveau, X ferme tout le panel droit.
- Les actions destructrices (archiver, supprimer, corriger) ramènent à la liste de la spec.
- Échap suit la même hiérarchie (change → liste → fermé).
- La liste de la spec garde son scroll et signale le dernier change consulté.

**Non-Goals:**
- Refléter la spec ou le change sélectionné dans l'URL (la spec impose un état local ; `?spec=` reste lu uniquement à l'initialisation).
- Modifier le comportement du `DetailPanel` dans le Kanban.
- Changer le comportement du clic sur la ligne de la spec déjà sélectionnée dans la grille (toujours : désélection).
- Retravailler le contenu de `SpecHistoryView` (barre de stats « 1 specs • N changes », largeur `max-w-3xl`).

## Decisions

**1. Props optionnelles sur `DetailPanel` : `onBack`, `backLabel`, `onActionDone`.**
- Si `onBack` est fourni, le header affiche au-dessus du titre une ligne cliquable « ← {backLabel} » (icône `ArrowLeft` de lucide-react, `aria-label` i18n « Retour à {{label}} »).
- `onClose` reste le callback du X. `onActionDone` est appelé après archivage, suppression et correction ; il vaut `onClose` par défaut, ce qui laisse le Kanban strictement inchangé.
- Dans `TimelinePage` : `onBack` et `onActionDone` ramènent à la liste de la spec, `onClose` désélectionne change et spec.
- Alternatives écartées : (B) masquer le X quand `onBack` est présent, ce qui impose deux clics pour tout fermer et surprend ; (C) garder le X comme retour, ce qui duplique le bouton retour sans lever l'ambiguïté. Un composant de panneau dédié à la Matrice duplique les 600 lignes de `DetailPanel`, ce qui est disproportionné.

**2. Liste de la spec conservée sous le `DetailPanel` plutôt que démontée ou masquée en `display: none`.**
- Le slot droit devient un conteneur `relative` : l'en-tête de spec + `SpecHistoryView` restent rendus en permanence ; le `DetailPanel` est superposé (`absolute inset-0`, fond blanc). La liste dessous est marquée `inert` et `aria-hidden` pour éviter le focus clavier et la lecture d'écran.
- Pourquoi : le scroll interne de `SpecHistoryView` (Radix ScrollArea) est préservé sans code de sauvegarde. `display: none` risque de réinitialiser `scrollTop` selon les navigateurs, et sauvegarder/restaurer manuellement le scroll ajoute un état fragile.
- Alternative écartée : démonter et restaurer le scroll depuis une ref, plus de code pour un résultat moins fiable.

**3. Dernier change consulté : état `lastViewedChange` dans `TimelinePage`.**
- Renseigné à l'ouverture d'un change ; transmis à `SpecHistoryView` via `selectedChangeName` (le surlignage existe déjà, pas de nouveau composant). Réinitialisé au changement de spec, à la fermeture complète ou au passage à l'onglet Changes.
- `selectedChange` (change ouvert) et `lastViewedChange` (change surligné) restent deux états : le premier pilote l'affichage, le second survit au retour.

**4. Échap géré dans `TimelinePage`, uniquement quand `mode === 'matrice'` et `selectedSpec` est non nul.**
- Écouteur `keydown` sur `document`, enregistré par `useEffect`. Si `selectedChange` : retour à la liste ; sinon : fermeture du panel.
- Ignoré si l'événement est déjà `defaultPrevented`, si la cible est un `input`, `textarea`, `select` ou un élément `contenteditable`, ou si un `[role="dialog"]` est présent dans le document (couvre `ApproveDialog`, `CorrectionDialog` et les dialogs Radix, qui gèrent leur propre Échap).
- Alternative écartée : écouteur React `onKeyDown` sur le panel, qui ne reçoit rien tant que le focus n'est pas dedans.

**5. Libellés.** `backLabel` est le nom de la spec (pas traduit). Le texte d'accessibilité passe par la clé `detailPanel:backTo` fr/en.

## Risks / Trade-offs

- [Superposition : le contenu de la liste reste dans le DOM et dans la mise en page] → `inert` + `aria-hidden`, et un test vérifiant qu'un élément de la liste n'est pas atteignable au clavier pendant le drill-down.
- [Échap peut entrer en conflit avec d'autres écouteurs globaux (dialogs custom basés sur `document.keydown`)] → la détection `[role="dialog"]` et `defaultPrevented` ; test couvrant l'ouverture de `ApproveDialog`/`CorrectionDialog`.
- [`DetailPanel` gagne 3 props optionnelles] → comportement par défaut identique, couvert par les tests existants du Kanban.
- [`inert` : support navigateur récent, typage React variable] → à vérifier dans la version de React du projet ; repli sur `aria-hidden` + `tabIndex={-1}` si besoin.

## Open Questions

- Aucune qui change la spec ou le découpage en tâches.
