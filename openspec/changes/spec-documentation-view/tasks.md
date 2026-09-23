# Tasks

## 1. Backend — Prompt de génération et invocation d'agent

- [ ] 1.1 Ajouter le template de prompt opensp8c (page list fixe, règles de skip, conventions Mermaid, chemin de sortie `docs/opensp8c/`) sous `backend/internal/agents/` et vérifier qu'il est chargé par un test unitaire dédié
- [ ] 1.2 Ajouter une fonction assemblant le prompt final (formalisme + contenu concaténé de `openspec/specs/**/spec.md`) et vérifier par test unitaire que la concaténation inclut bien tous les fichiers `spec.md` d'un répertoire de test
- [ ] 1.3 Créer `backend/internal/api/handlers/docs.go` avec un `DocsHandler` suivant le pattern de `FFHandler` (garde de concurrence par workspace id via une map `running`) et vérifier par test que deux déclenchements concurrents pour le même workspace renvoient un `409` sur le second
- [ ] 1.4 Implémenter le lancement du run en réutilisant `agents.AgentConfig.BuildSubprocessArgs` avec le prompt de 1.2 comme `extraPrompt`, en écrivant la sortie sous `<workspace path>/docs/opensp8c/` et vérifier par test avec un agent factice (binaire de test) que les fichiers attendus sont créés au bon chemin

## 2. Backend — Endpoints de consultation et fraîcheur

- [ ] 2.1 Ajouter `GET /api/workspaces/{id}/docs` listant les pages présentes sous `docs/opensp8c/` dans l'ordre fixe (overview, architecture, domain-model, workflows), en filtrant les pages absentes, et vérifier par test que l'ordre et le filtrage sont corrects
- [ ] 2.2 Ajouter le calcul de `is_stale` dans cette réponse (comparaison de la mtime la plus récente parmi `openspec/specs/**/spec.md` et celle des pages sous `docs/opensp8c/`) et vérifier par test les trois scénarios de la spec `spec-documentation-generation` (spec plus récente, doc à jour, aucune page générée)
- [ ] 2.3 Ajouter `GET /api/workspaces/{id}/docs/{page}` retournant le contenu Markdown brut d'une page, avec 404 si la page n'existe pas, et vérifier par test
- [ ] 2.4 Ajouter `POST /api/workspaces/{id}/docs/generate` déclenchant 1.3/1.4, retournant `202` au démarrage et `409` si un run est déjà en cours, et vérifier par test
- [ ] 2.5 Enregistrer les nouvelles routes dans `backend/internal/api/router.go` aux côtés des routes `specs` existantes et vérifier par `go build ./...` que le routeur compile

## 3. Backend — Événements de progression

- [ ] 3.1 Émettre `docs_generation_started` au lancement du run et `docs_generation_done` / `docs_generation_failed` à sa fin sur le flux SSE existant `/api/workspaces/{id}/events` (`internal/watcher`), et vérifier par test que les événements sont bien publiés dans les trois cas (succès, échec, déjà en cours → pas d'événement dupliqué)

## 4. Frontend — Sous-onglets Spécifications / Documentation

- [ ] 4.1 Ajouter la navigation par sous-onglets dans `SpecsPage.tsx` ("Spécifications" par défaut, "Documentation" nouveau), en déplaçant le rendu actuel (liste + contenu + TOC) sous le sous-onglet "Spécifications" sans changer son comportement, et vérifier manuellement que la vue existante reste identique
- [ ] 4.2 Créer un hook `useDocs(workspaceId)` (React Query) appelant `GET /api/workspaces/{id}/docs` et un hook `useDocPage(workspaceId, page)` appelant `GET /api/workspaces/{id}/docs/{page}`, dans `frontend/src/hooks/`, et vérifier par un test de hook ou un appel manuel que les données attendues sont retournées
- [ ] 4.3 Créer le panneau du sous-onglet "Documentation" (liste des pages existantes, sélection, état vide avec bouton "Générer" si aucune page) et vérifier manuellement l'affichage sur un workspace sans documentation générée

## 5. Frontend — Rendu Mermaid

- [ ] 5.1 Ajouter la dépendance `mermaid` au `package.json` du frontend et vérifier que l'installation aboutit
- [ ] 5.2 Créer un composant de rendu Mermaid branché sur `components.code` de `react-markdown` (chargement paresseux, actif uniquement dans le sous-onglet "Documentation"), avec repli sur un bloc de code brut en cas d'échec de parsing, et vérifier manuellement avec un diagramme valide et un diagramme invalide

## 6. Frontend — Déclenchement, état d'exécution et badge de fraîcheur

- [ ] 6.1 Ajouter le bouton "Générer" au sous-onglet "Documentation" appelant `POST .../docs/generate`, désactivé pendant un run en cours et vérifier manuellement qu'un second clic pendant l'exécution ne déclenche pas de second run
- [ ] 6.2 Étendre `useWorkspaceLiveState.ts` (ou ajouter un hook dédié suivant le même pattern) pour écouter `docs_generation_started` / `docs_generation_done` / `docs_generation_failed`, mettre à jour l'état "en cours" et invalider la query `useDocs` à la fin, et vérifier manuellement que la liste des pages se rafraîchit automatiquement après une génération
- [ ] 6.3 Afficher le badge "documentation potentiellement obsolète" dans le sous-onglet "Documentation" à partir du champ `is_stale`, et vérifier manuellement son apparition/disparition en modifiant un fichier spec puis en régénérant

## 7. i18n

- [ ] 7.1 Ajouter les clés du sous-onglet Documentation (titre, état vide, bouton Générer, état en cours, badge obsolète) dans `frontend/src/locales/fr/specs.json` et `frontend/src/locales/en/specs.json`, et vérifier qu'aucune clé n'est manquante (cohérence FR/EN)

## 8. Vérification bout-en-bout

- [ ] 8.1 Sur un workspace de test avec plusieurs capacités `openspec/specs/`, déclencher une génération complète et vérifier que `docs/opensp8c/overview.md`, `architecture.md` et `domain-model.md` sont créés, que `workflows.md` n'est créé que si un cycle de vie multi-étapes est présent, et qu'aucun fichier existant sous `docs/` n'est modifié
- [ ] 8.2 Exécuter la suite de tests backend (`go test ./...`) et frontend (tests existants) et vérifier qu'ils passent
