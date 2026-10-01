# Tasks

## 1. Libellés i18n

- [ ] 1.1 Ajouter les clés `docs.diagram.enlarge`, `docs.diagram.close` et `docs.diagram.modalTitle` dans `frontend/src/locales/fr/specs.json` et `frontend/src/locales/en/specs.json` ; vérifier que les deux fichiers restent du JSON valide avec les mêmes clés (`node -e "JSON.parse(require('fs').readFileSync('src/locales/fr/specs.json'))"` depuis `frontend/`, idem en)

## 2. Agrandissement dans `MermaidDiagram`

- [ ] 2.1 Dans `frontend/src/components/MermaidDiagram.tsx`, envelopper le SVG rendu dans un `<button type="button">` (`cursor-zoom-in`, anneau de focus visible, `aria-label` = `docs.diagram.enlarge`) qui ouvre un état local `open` ; les branches `failed` (bloc `<pre><code>`) et `svg === null` restent inchangées et sans bouton ; vérifier que `npx tsc -b` passe depuis `frontend/`
- [ ] 2.2 Ajouter la modale avec `@radix-ui/react-dialog` (`Dialog.Root` contrôlé, `Dialog.Portal`, `Dialog.Overlay`, `Dialog.Content` d'environ 90vw × 90vh, `Dialog.Title` en `sr-only`, `aria-describedby={undefined}`, bouton de fermeture `docs.diagram.close`), injectant le même SVG sans second `mermaid.render`, avec une classe dédiée forçant `svg { width: 100%; height: 100%; max-width: none !important }` pour agrandir y compris les petits diagrammes ; stopper la propagation des clics de la modale vers le Markdown ; vérifier visuellement en lançant l'application (skill `run`) sur `architecture.md` : clic ouvre la modale avec un schéma plus grand qu'en ligne, Échap / croix / fond la ferment, la position de défilement et la page sélectionnée sont conservées
- [ ] 2.3 Créer `frontend/src/components/MermaidDiagram.test.tsx` (mock de `mermaid` via `vi.mock('mermaid', ...)` renvoyant un SVG fixe, rendu avec `@testing-library/react` si disponible, sinon avec le harnais déjà utilisé par `AgentPoolModal.test.tsx`) couvrant : le diagramme rendu est un bouton avec le libellé accessible ; le clic ouvre une modale contenant le SVG ; Échap ferme la modale ; le bouton de fermeture ferme la modale ; un échec de rendu affiche le bloc de code brut sans bouton ni modale ; vérifier avec `npx vitest run src/components/MermaidDiagram.test.tsx` depuis `frontend/`

## 3. Vérification d'intégration

- [ ] 3.1 Lancer l'application (skill `run`), ouvrir un workspace, onglet Specs > Documentation, et sur chaque page de `docs/opensp8c/` contenant un diagramme vérifier : curseur zoom-in au survol, ouverture au clic et au clavier (Tab puis Entrée / Espace), modale ne montrant que le diagramme cliqué, fermeture par Échap / croix / fond, retour du focus sur le diagramme après fermeture ; vérifier aussi qu'un petit diagramme est agrandi au-delà de sa taille naturelle dans la modale et qu'un bloc Mermaid invalide (à simuler dans une page temporaire) reste un bloc de code non cliquable
- [ ] 3.2 Vérifier que le sous-onglet « Spécifications » est inchangé, puis lancer `cd frontend && npx vitest run && npm run lint && npm run build` sans erreur ; si `docs-toc-wide-reading` a été intégré entre-temps, vérifier que le style en ligne de `.mermaid-diagram` (max-width / overflow) s'applique toujours dans le bouton
