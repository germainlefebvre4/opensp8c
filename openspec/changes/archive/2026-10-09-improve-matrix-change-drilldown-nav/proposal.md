# Proposal

## Why

Dans la TimelinePage, onglet Matrice, un clic sur une spec ouvre un panneau droit listant ses changes ; un clic sur l'un de ces changes remplace ce panneau par le `DetailPanel` du change. Ce `DetailPanel` est le composant du Kanban et n'a qu'un bouton « X ». Ici le X sert en réalité de « retour à la liste de la spec », et la spec `timeline-spec-matrix` le décrit ainsi (« fermer le DetailPanel revient à la liste »). Résultat : l'utilisateur s'attend à un bouton retour, ne trouve qu'une croix qui ressemble à « tout fermer », et le panneau n'indique plus à quelle spec le change est rattaché. Au retour, la liste est remontée de zéro (scroll perdu), ce qui pénalise les specs avec beaucoup de changes.

## What Changes

- **Bouton retour nommé** : en drill-down depuis la Matrice, le header du `DetailPanel` affiche une ligne « ← nom-de-la-spec » au-dessus du titre du change ; un clic ramène à la liste des changes de la spec.
- **Le X ferme tout** dans ce contexte : le change et la spec sont désélectionnés, la grille reprend toute la largeur (même sémantique que le X du Kanban).
- **Actions destructrices** : après archivage, suppression ou demande de correction depuis le drill-down, le panneau revient à la liste de la spec (et non à la fermeture complète).
- **Échap** : dans le drill-down, Échap revient à la liste de la spec ; dans la liste de la spec, Échap ferme le panneau. Échap est ignoré tant qu'une boîte de dialogue ou un champ de saisie a le focus.
- **Contexte de la liste conservé** : la liste des changes de la spec reste montée (masquée) pendant le drill-down pour préserver le scroll ; au retour, le dernier change consulté est légèrement mis en évidence.
- Le Kanban reste inchangé : le `DetailPanel` n'affiche le retour que si on lui fournit un callback `onBack`.

## Capabilities

### New Capabilities

### Modified Capabilities
- `timeline-spec-matrix`: le requirement « Drill-down vers le DetailPanel depuis le panel de spec » est modifié (retour explicite via « ← spec », X qui ferme tout, retour après action destructrice, Échap, conservation du scroll et du dernier change consulté).

## Impact

- Frontend : `components/DetailPanel.tsx` (props optionnelles `onBack` / `backLabel`, ligne de retour, callback distinct pour les actions post-archivage/suppression/correction), `pages/TimelinePage.tsx` (callbacks, liste masquée plutôt que démontée, gestion d'Échap, état du dernier change consulté), `components/SpecHistoryView.tsx` (mise en évidence du dernier change consulté), locales fr/en (`timeline.json`, `detailPanel.json`), tests associés (`DetailPanel.test.tsx`, test de la TimelinePage).
- Aucun impact backend ni API. Aucun changement de comportement côté Kanban.
