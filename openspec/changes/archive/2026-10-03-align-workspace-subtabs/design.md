# Design

## Context

Quatre implémentations inline de sous-onglets `border-b-2` existent dans le frontend. SpecsPage et TimelinePage dupliquent le même helper `subTabClass` (`px-3 py-2`, `border-slate-200`, `bg-white`, sans titre). SettingsPage et ConfigurationPage utilisent une seconde variante (`px-6 pt-3 gap-4`, `border-slate-100`) sous un `<h1>`. AgentsPage n'a pas de sous-onglets. Voir proposal.md pour la motivation.

Contraintes observées :
- `WorkspaceTabs` recopie toute la query string d'un onglet principal à l'autre : `tab` et `run` « fuient » donc vers les autres pages du workspace.
- Specs et Timeline gardent leur sous-onglet en état local (le spec `timeline-spec-matrix` l'exige pour Timeline). Settings le garde dans `?tab=`.
- `AgentRunPanel` est piloté par `?run=<change>/<ts>`, et ses appels `onSelectRun`/`onClose` modifient l'URL.

## Goals / Non-Goals

**Goals:**
- Un composant `SubTabs` unique, utilisé par Specs, Timeline, Settings et Agents, au rendu identique à la barre actuelle de Specs/Timeline.
- Découpage d'Agents en Workers / Runs sans perdre la sélection de run ni les liens existants.

**Non-Goals:**
- Migrer ConfigurationPage, ou changer `WorkspaceTabs`.
- Changer la persistance en URL de Specs ou Timeline.
- Empêcher la fuite de `tab`/`run` entre onglets principaux (comportement existant de `WorkspaceTabs`).

## Decisions

**1. `SubTabs` est un composant contrôlé et sans routage.** Props : `tabs: {id, label, testId?}[]`, `active: id`, `onChange(id)`, `trailing?: ReactNode`. Il rend la barre actuelle de Specs/Timeline (`shrink-0 flex items-center gap-1 px-4 border-b border-slate-200 bg-white`) et pose `trailing` dans un conteneur `ml-auto`. Chaque page garde la responsabilité de l'état (local pour Specs/Timeline, URL pour Settings/Agents).
Alternative : un composant qui gère lui-même `?tab=`. Rejeté : Specs et Timeline doivent rester locaux, et le défaut/validation de valeur diffère par page.

**2. Le nom du workspace de Settings va dans `trailing`, pas supprimé.** Le spec `workspace-settings` exige que l'écran affiche le nom du workspace ; retirer le `<h1>` ne doit pas le faire disparaître. L'élément de `trailing` est un `<span>` texte, non interactif (`text-[11px] text-slate-500`, repris de l'existant).
Alternative : supprimer le nom. Rejeté : il faudrait retirer l'exigence du spec sans que la demande y conduise.

**3. Résolution de l'onglet actif d'Agents : un `tab` valide d'abord, sinon `run` ⇒ Runs, sinon Workers.** C'est ce qui permet à la fois « `?run=` seul ouvre Runs » (liens existants, rechargement) et « cliquer un worker depuis Workers reste sur Workers ». Pour cela :
- `selectRun` appelé depuis une ligne de worker écrit `run=…` **et `tab=workers`** ;
- `selectRun` appelé depuis une ligne de run écrit `run=…` et `tab=runs` ;
- `selectRun` appelé par `AgentRunPanel` (sélecteur de run d'un même change) ne touche pas à `tab` ;
- `closePanel` supprime `run` et garde `tab` ;
- cliquer « Workers » supprime `tab` seulement si aucun `run` n'est présent, sinon écrit `tab=workers` explicitement (sinon `run` rebasculerait sur Runs) ; cliquer « Runs » écrit `tab=runs`.
Alternative : `run` force toujours Runs. Rejeté : un clic sur un worker ferait sauter l'utilisateur d'onglet sous ses yeux.

Une fonction pure `parseAgentsTab(tabParam, runParam)` (à côté de `parseRunSelection` dans `lib/poolRuns.ts`, ou dans la page comme `parseSettingsTab`) porte cette règle et se teste seule.

**4. Les valeurs de `tab` d'Agents et de Settings restent disjointes.** Comme `WorkspaceTabs` propage `tab`, une valeur de Settings (`columns`…) arrivant sur Agents est ignorée (inconnue ⇒ règle 3), et inversement. Cette disjonction est la seule protection ; elle est notée en commentaire près des deux parseurs.

**5. Le panneau de run reste dans la mise en page actuelle**, commun aux deux sous-onglets : la page garde la rangée `flex` « colonne principale + `AgentRunPanel` », et seul le contenu de la colonne principale change selon le sous-onglet. La table des workers et la table des runs restent inchangées, seulement réparties.

**6. Libellés.** Nouvelles clés `tabs.workers` et `tabs.runs` dans les locales `agents` fr et en. Les clés `title` existantes (`agents`, `settings`) ne sont plus affichées en `<h1>` ; elles sont conservées pour l'`aria-label` de la barre de sous-onglets afin de garder un nom accessible. Les tests de parité de locales (`*.i18n.test.ts`) restent verts.

## Risks / Trade-offs

- [Conflit de fichier avec `improve-matrix-change-drilldown-nav` sur `TimelinePage.tsx`] → le remplacement de la barre est localisé (helper `subTabClass` et bloc de boutons) ; faire ce change après, ou rebaser, et ne pas toucher au reste du fichier.
- [Tests Agents existants supposent les runs visibles par défaut (`run-row`, texte « Aucun run… »)] → ils doivent d'abord ouvrir `?tab=runs`; c'est le changement de comportement voulu, pas une régression.
- [Règle d'URL d'Agents plus subtile que celle de Settings] → couverte par un test unitaire de `parseAgentsTab` et par des tests d'interaction (clic worker, clic run, lien `run=` seul, `tab` inconnu).
- [Perte de repère visuel du titre de page] → la page active reste indiquée par l'onglet principal du workspace (`WorkspaceTabs`).
