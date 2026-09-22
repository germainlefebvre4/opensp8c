# Tasks

## 1. Fix the Kanban column key-casing bug

- [x] 1.1 In `pages/KanbanPage.tsx`, change `t('columns.toexplore')` → `t('columns.toExplore')`, `t('columns.todo')` → `t('columns.toDo')`, `t('columns.inprogress')` → `t('columns.inProgress')`, and add the `columns.toReview` key to `en/kanban.json` and `fr/kanban.json` (removing the inline `defaultValue`). Verify by running the app, toggling EN/FR, and confirming all 4 leading Kanban columns show real translated text, not a raw key or English leaking into French.

## 2. Locale namespace groundwork

- [x] 2.1 Add a new `timeline` namespace: create `src/locales/en/timeline.json` and `src/locales/fr/timeline.json`, and register `timeline` in `src/i18n.ts`'s `ns` array, imports, and `resources.en`/`resources.fr` maps, following the exact pattern of the other 8 namespaces. Verify by running the app and confirming no i18next "namespace not loaded" console warnings appear on the Timeline page.
- [x] 2.2 Add the new key groups to existing namespace files per design.md: `kanban.json` (`agentPoolButton`, `promoteGhostDialog`, `deleteGhostDialog`, `columnActions`, `card`), `dialogs.json` (`agentPool`), `detailPanel.json` (`ghostBanner`, `reviewActions`, solidify-error key), `workspace.json` (`agentSelector`), `specs.json` (`editor`, `history`, `toc`, `viewHistoryLink`), `explore.json` (`draftPanel`, `typingBubble`) — in both `en` and `fr`, with real (non-duplicated) French text. Verify with `node -e "JSON.parse(require('fs').readFileSync(f))"` (or equivalent) on every edited file to confirm valid JSON, and a diff review confirming every `en` key added has a matching `fr` key.

## 3. Kanban board components

- [x] 3.1 Wire `pages/KanbanPage.tsx`'s Agent Pool toggle button, promote-ghost dialog, and delete-ghost dialog to the new `kanban.json` keys (from task 2.2) and `tCommon('cancel')` for both dialogs' Cancel buttons, removing all hardcoded French strings. Verify by opening both dialogs and the pool button in EN and FR and confirming correct text in each.
- [x] 3.2 Wire `components/KanbanColumn.tsx` tooltips/buttons ("Nouvelle exploration", "Afficher/Réduire la colonne", "Afficher plus (N)") to `kanban.json`'s `columnActions` keys via `useTranslation('kanban')`. Verify by hovering/toggling each control in both languages.
- [x] 3.3 Wire `components/ChangeCard.tsx`'s hardcoded fragments ("Figer", "projet", "Sync & Archive", "tasks", "exploring", `ff...`/`ff failed`, "dft") to `kanban.json`'s `card` keys, using interpolation for the task-count strings. Verify by rendering cards in each Kanban status (todo, in-progress, to-review, done, ghost/exploring) in both languages.

## 4. Agent Pool and Agent Selector

- [x] 4.1 Wire `components/AgentPoolModal.tsx` (heading, body copy, worker-count label, both delegation-mode labels/descriptions, cancel/start buttons) to the new `dialogs.json` `agentPool` keys. Verify by opening the modal in both languages and checking every string, including the pluralized `{size} Agent(s)` label via i18next `count` interpolation.
- [x] 4.2 Wire `components/AgentSelector.tsx` tooltips ("Choisir l'agent de code", "Configurer les variables d'environnement de l'agent") and the "non installé" status label to `workspace.json`'s `agentSelector` keys. Verify by inspecting the sidebar agent selector in both languages, including the not-installed state.

## 5. Detail panel

- [x] 5.1 Wire `components/DetailPanel.tsx`'s ghost-draft banner ("Brouillon d'exploration actif", "Figer le change"), review-tab buttons ("Approuver & Fusionner", "Demander correction"), and the solidify error message to `detailPanel.json`'s `ghostBanner`/`reviewActions` keys. Verify by opening a ghost-linked change and a to-review change in both languages and confirming all four strings and the error path render translated text.

## 6. Specs feature

- [x] 6.1 Wire `components/SpecEditor.tsx` (unsaved-changes banner, Ignorer/Recharger/Annuler buttons, save error, "Aucune modification", Enregistrement/Enregistrer) to `specs.json`'s new `editor` keys, reusing `tCommon('cancel')`/`tCommon('save')` where design.md calls for reuse. Verify by editing a spec, triggering the external-modification banner and a save error, in both languages.
- [x] 6.2 Wire `components/SpecHistoryView.tsx` (untraced/orphan counts, "aucun change lié", "actif" badge, "Cette spec n'est liée à aucun change", orphan explainer) to `specs.json`'s new `history` keys, using pluralization interpolation for the count-based strings. Verify by viewing spec history for a spec with 0, 1, and >1 linked changes, and the orphans view, in both languages.
- [x] 6.3 Wire `components/TableOfContents.tsx`'s "Sur cette page" heading to `specs.json`'s `toc.title`, and `pages/SpecsPage.tsx`'s "Voir l'historique →" link to `specs.json`'s `viewHistoryLink`. Verify by viewing a spec page's table of contents and history link in both languages.

## 7. Timeline feature

- [x] 7.1 Wire `pages/TimelinePage.tsx` (page heading, "Changes"/"Matrice" tabs, "Specs fréquentes" label, "Tout effacer" button, empty-state message, loading state, "Voir la spec →" link, and the `MONTHS` abbreviation array) to the new `timeline` namespace, using i18next's date-formatting facilities or a per-locale month array so month abbreviations render correctly in both languages. Verify by browsing the Timeline page and its date-grouped headings in both languages.
- [x] 7.2 Wire `components/TimelineSpecMatrix.tsx` (`GRANULARITY_OPTIONS` Jour/Semaine/Mois/Trimestre, "Spec × période" label, loading state, "Spec" header, change-count tooltip, "Orphelins — ..." explainer) to the `timeline` namespace. Verify by switching granularity options and viewing the matrix in both languages.
- [x] 7.3 Wire `components/TimelineChangeCard.tsx`'s "Voir spec: {{spec}}" tooltip to the `timeline` namespace with interpolation. Verify by hovering a timeline change card's spec link in both languages.

## 8. Explore / draft experience

- [x] 8.1 Wire `components/DraftSidePanel.tsx` (loading/error states, "Brouillon de Change" header, save-status indicators, save tooltip, description placeholder, "Tâches Déduites (N)", "Ajouter" button, empty state, task input placeholder) to `explore.json`'s new `draftPanel` keys, using interpolation for the task count. Verify by opening an anonymous exploration's draft panel, triggering each save state, in both languages.
- [x] 8.2 Wire `components/TypingBubble.tsx`'s "{{name}} réfléchit..." string to `explore.json`'s new `typingBubble` key using i18next interpolation for the agent name (replacing string concatenation). Verify by triggering the typing indicator with both a named agent and the "Claude" fallback, in both languages.

## 9. Final verification

- [x] 9.1 Re-run the audit's detection method: grep the touched files (and the rest of `frontend/src`, excluding `locales/`) for accented characters and common French UI words (`Annuler`, `Créer`, `Abandonner`, `Supprimer`, `Enregistrer`, `Fermer`, `Chargement`, `Confirmer`, etc.) to confirm no hardcoded French strings remain outside `src/locales/`. Verify by the grep returning no matches in the files listed in this change's Impact section.
- [x] 9.2 Re-run the casing-mismatch check: for every `t('...')` / `t("...")` call in `frontend/src`, confirm the key exists (matching casing) in at least one locale namespace file. Verify by the check reporting zero mismatches (the same method used in the original audit, adapted as a one-off script or manual review).
- [x] 9.3 Manually toggle the language switcher through the whole app (Kanban board incl. both dialogs and Agent Pool modal, Detail Panel incl. a to-review and a ghost-linked change, Specs page incl. editor and history view, Timeline page incl. matrix, an anonymous exploration with the draft panel open) in both EN and FR. Verify no hardcoded-looking text, no raw translation keys, and no English leaking into the French pass (or vice versa).
