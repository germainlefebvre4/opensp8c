# Tasks

## 1. Dépendances

- [x] 1.1 Ajouter `@radix-ui/react-dialog` et `@radix-ui/react-toast` à `frontend/package.json` et vérifier que `npm install` (ou équivalent) s'exécute sans erreur

## 2. Composant `ConfirmDialog` générique

- [x] 2.1 Créer `frontend/src/components/ui/ConfirmDialog.tsx` basé sur `@radix-ui/react-dialog`, avec props `open`, `title`, `body`, `confirmLabel`, `cancelLabel`, `variant` (`destructive` par défaut ici), `onConfirm`, `onCancel`, `isPending`, `error`, `onRetry` — inspiré du pattern visuel de `ResetTasksDialog.tsx` (icône d'alerte, boutons rouge destructif) et vérifier qu'il rend un `role="dialog"`/`aria-modal` et se ferme sur Escape ou clic sur l'overlay
- [x] 2.2 Gérer dans le composant les 3 états visuels : confirmation (boutons actifs), en cours (spinner + texte "Synchronisation et archivage en cours...", boutons désactivés, fermeture bloquée), erreur (message inline + bouton "Réessayer", dialog reste ouvert) et vérifier chaque état via `npm run build` (typecheck) puis test manuel dans le navigateur
- [x] 2.3 Bloquer la fermeture du dialog (Escape, clic overlay, bouton Annuler) pendant l'état `isPending` et vérifier manuellement qu'aucune interaction ne ferme le dialog pendant l'appel en cours

## 3. Système de toast générique

- [x] 3.1 Créer `frontend/src/components/ui/Toast.tsx` + provider (`ToastProvider`) basé sur `@radix-ui/react-toast`, monté une fois au niveau racine de l'app (`App.tsx` ou équivalent), et vérifier qu'il ne casse pas le rendu existant (`npm run build`)
- [x] 3.2 Créer le hook `frontend/src/hooks/useToast.ts` exposant une fonction du type `toast({ title, variant })`, utilisable depuis n'importe quel composant, et vérifier via un appel de test manuel (toast temporaire) qu'il s'affiche puis se referme automatiquement

## 4. Libellés i18n

- [x] 4.1 Ajouter les clés `archive`/`archiving`/`archiveConfirmTitle`/`archiveConfirmBody`/`archiveSuccessToast` (noms à ajuster selon convention du fichier) dans `frontend/src/locales/fr/*.json` et `frontend/src/locales/en/*.json`, en unifiant sur le libellé "Sync & Archive" pour le bouton déclencheur aux deux endroits, et vérifier que les deux fichiers restent des JSON valides
- [x] 4.2 Retirer le texte en dur "Sync & Archive" de `ChangeCard.tsx` (ligne ~237) au profit de la clé i18n ajoutée en 4.1, et vérifier via `npm run lint`

## 5. Intégration carte Kanban (`ChangeCard.tsx`)

- [x] 5.1 Remplacer l'appel direct à `handleArchive`/`archive.mutateAsync` au clic par l'ouverture du `ConfirmDialog` (état local `confirmOpen`), et vérifier manuellement que le clic sur "Sync & Archive" n'appelle plus l'API avant confirmation
- [x] 5.2 Câbler la confirmation du dialog sur `archive.mutateAsync(change.name)` existant, et le succès sur la fermeture du dialog + déclenchement du toast (`useToast`) + invalidation des queries déjà en place dans `useArchive.ts` (aucun changement requis dans `useArchive.ts`), et vérifier manuellement le flux complet (clic → confirmation → succès → toast → carte disparue)
- [x] 5.3 Câbler l'échec de la mutation sur l'état d'erreur du dialog (au lieu de l'affichage inline actuel sur la carte, lignes ~222-231), avec bouton "Réessayer" qui relance `archive.mutateAsync`, et vérifier manuellement qu'une erreur simulée (ex. change avec tasks non finalisées) laisse le dialog ouvert avec le message

## 6. Intégration DetailPanel (`DetailPanel.tsx`)

- [x] 6.1 Remplacer l'appel direct à `handleArchive` (lignes ~54-63) par l'ouverture du même `ConfirmDialog` partagé, avec le même libellé unifié que la carte Kanban, et vérifier manuellement que le clic sur le bouton n'appelle plus l'API avant confirmation
- [x] 6.2 Câbler confirmation → succès → toast → fermeture du DetailPanel (`onClose()` existant, ligne ~58), et erreur → dialog reste ouvert avec message + retry (au lieu de l'affichage inline actuel, lignes ~399-401), et vérifier manuellement le flux complet depuis le DetailPanel

## 7. Vérification globale

- [x] 7.1 Lancer `npm run build` et `npm run lint` dans `frontend/` et vérifier qu'ils passent sans erreur
- [x] 7.2 Démarrer l'app en local, archiver un change de test depuis la carte Kanban ET depuis le DetailPanel, et vérifier dans le navigateur : confirmation avant l'appel, spinner pendant l'appel, toast de succès, disparition de la carte, mise à jour de la colonne Archived
- [x] 7.3 Vérifier manuellement le cas d'annulation (clic sur "Annuler" ou Escape) : aucun appel réseau déclenché (via les outils réseau du navigateur), dialog fermé, carte/DetailPanel inchangés
