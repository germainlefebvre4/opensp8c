# Design

## Context

Aujourd'hui, `ChangeCard.tsx` et `DetailPanel.tsx` appellent chacun directement `useArchive(workspaceId).mutateAsync(change.name)` au clic, sans confirmation. Le hook `useArchive` (`frontend/src/hooks/useArchive.ts`) fait un seul `POST /api/workspaces/{id}/changes/{name}/archive`, géré côté backend par `archive.go` qui exécute `openspec archive <name> --yes` (une seule commande CLI bloquante, sync + archive inclus). En cas de succès, `onSuccess` invalide les queries `changes` et `archived-changes` ; React Query refetch et la carte disparaît dès que la liste revient sans elle — sans animation, sans confirmation, sans message de succès.

Le projet n'a aucun composant Dialog/Toast générique. Il existe 4 implémentations de modal ad hoc dupliquées (`ResetTasksDialog.tsx` étant la plus proche d'un modal de confirmation : icône d'alerte, titre, texte, boutons Annuler/Confirmer). Radix est déjà utilisé pour d'autres primitives (`@radix-ui/react-scroll-area`, `-separator`, `-tooltip`) mais pas `react-dialog` ni `react-toast`. Aucun des modals existants ne gère Escape, focus trap, ou `aria-modal`. Le thème n'a pas de mode dark (`0` occurrence de `dark:` dans `src/`), et les couleurs destructives/warning utilisent directement `red-*`/`amber-*` (pas de remapping via `@theme`), avec `lucide-react` pour les icônes.

Voir `proposal.md` pour le "pourquoi" ; voir `specs/change-archive/spec.md` et `specs/kanban-change-detail/spec.md` pour le comportement attendu.

## Goals / Non-Goals

**Goals:**
- Un seul `ConfirmDialog` générique, accessible (Escape, focus trap, `aria-modal`), réutilisé par les deux points d'entrée (carte Kanban, DetailPanel).
- Un système de toast générique (provider + hook), posé une fois, réutilisable par de futures actions.
- Aucune régression du flux d'archivage existant (même endpoint, même commande CLI, même gestion d'erreur avec retry).

**Non-Goals:**
- Pas de changement du contrat API/CLI backend : `openspec archive <name> --yes` reste un appel unique et bloquant. La progression affichée ("Synchronisation et archivage en cours...") est une mise en scène front, pas un suivi réel d'étapes séparées côté serveur.
- Pas de vraie granularité de progression (sync vs archive) : décision explicite du porteur de la demande, pour éviter le risque d'état intermédiaire (specs synchronisées mais change non archivé) qu'introduirait un découpage en deux appels.
- Pas de refonte des 3 autres modals ad hoc du projet (`AgentPoolModal.tsx`, `AgentSettingsModal.tsx`, le modal inline de `KanbanPage.tsx`) : ils ne sont pas dans le périmètre de cette change, même s'ils pourraient migrer vers `ConfirmDialog`/Radix plus tard.
- Pas de mode dark pour ces nouveaux composants (l'app n'en a pas).

## Decisions

**1. `@radix-ui/react-dialog` pour `ConfirmDialog`, plutôt que le pattern maison de `ResetTasksDialog.tsx`.**
Radix est déjà la lib de primitives du projet (scroll-area, separator, tooltip), et aucun des 4 modals existants ne gère Escape/focus trap/`aria-modal`. Pour une action irréversible (archivage), cette garantie d'accessibilité est justifiée. Alternative écartée : dupliquer le pattern `fixed inset-0 z-50 ...` de `ResetTasksDialog.tsx` — plus rapide à écrire mais reconduit un vrai manque d'accessibilité déjà identifié comme problème existant.

**2. Un seul `ConfirmDialog` générique, paramétrable (`title`, `body`, `confirmLabel`, `cancelLabel`, `variant`, `onConfirm`, `onCancel`, états `pending`/`error`), monté depuis `ChangeCard.tsx` et `DetailPanel.tsx`.**
Les deux points d'entrée déclenchent la même mutation `useArchive` (déjà partagée) ; leur factoriser aussi le dialog évite de dupliquer la logique de confirmation/progression/erreur, et résout au passage l'incohérence de libellé actuelle ("Sync & Archive" en dur sur la carte vs "Archive" traduit dans le DetailPanel) en unifiant sur "Sync & Archive", passé en i18n.

**3. Mise en scène front simple pour la progression ("Synchronisation et archivage en cours...") plutôt qu'un vrai suivi en 2 étapes.**
Le backend ne fait qu'un seul appel CLI bloquant. Séparer réellement sync et archive en deux appels introduirait un état intermédiaire à gérer (sync réussi, archive échoué) et changerait le contrat CLI/API actuel pour un bénéfice UX marginal. Décision explicite : rester sur un seul appel, avec un texte de progression générique côté dialog.

**4. Toast générique réutilisable, provider + hook `useToast()`, probablement via `@radix-ui/react-toast` pour rester cohérent avec le choix Radix du point 1.**
Aucun système de toast n'existe dans le projet ; en poser un réutilisable maintenant évite de refaire ce travail à la prochaine action qui en aura besoin. Alternative écartée : notification scopée à la seule action d'archivage (plus rapide à écrire mais à refaire ailleurs).

**5. Le libellé unifié reste "Sync & Archive" (et non "Archive"/"Archiver") aux deux endroits, en le passant par i18n.**
"Sync & Archive" décrit plus fidèlement ce que fait réellement `openspec archive --yes` (sync des specs incluse), et c'est déjà le libellé documenté dans `specs/kanban-board/spec.md` pour la carte Kanban - qui n'est pas modifié par cette change et reste donc la référence.

## Risks / Trade-offs

- **[Risque] Les specs principales `change-archive` et `kanban-change-detail` sont structurellement invalides** (`openspec validate --strict` : requirements hors de la section `## Requirements`, pas de `## Purpose`) → `openspec archive` refusera de merger les deltas de cette change tant que ces fichiers ne sont pas corrigés. C'est un problème préexistant, hors périmètre de cette change (pas de changement de comportement associé) ; il faudra le corriger séparément avant d'archiver cette change, ou dans le cadre de cette change si le porteur préfère l'inclure - à confirmer avant l'implémentation.
- **[Risque] Nouvelle dépendance (`@radix-ui/react-dialog`, `@radix-ui/react-toast`)** → limité par le fait que Radix est déjà utilisé dans le projet pour d'autres primitives ; pas de nouvelle lib à apprendre pour l'équipe.
- **[Trade-off] La progression affichée pendant l'appel est indicative, pas un vrai suivi d'étapes** → assumé (voir Goals/Non-Goals et Décision 3) ; si `openspec archive` devient long ou échoue tardivement après la phase de sync, l'utilisateur ne peut pas distinguer "sync en cours" de "archive en cours".
- **[Risque] Changement de comportement pour les utilisateurs habitués à l'archivage instantané** → mitigé par le fait que l'action reste rapide (un clic de confirmation supplémentaire), et que c'est justement le problème signalé (archivage accidentel sans confirmation).
