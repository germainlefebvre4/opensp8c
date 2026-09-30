# Tasks

## 1. Hook et calcul partagé

- [ ] 1.1 Ajouter à `frontend/src/hooks/usePoolStatus.ts` un `refetchInterval` de 3 s et un helper de calcul des workers disponibles (`max(0, size - workers.length)`) ; vérifier avec un test unitaire du helper (0 worker, partiel, plein, dépassement borné à 0)

## 2. Bouton d'en-tête Kanban

- [ ] 2.1 Afficher « actifs/taille » sur le bouton du Kanban lorsque `is_running`, rien sinon (`frontend/src/pages/KanbanPage.tsx`) ; ajouter les clés `fr`/`en` dans `kanban.json` si nécessaire et vérifier par un test de rendu (pool actif 0/3, pool actif 1/3, pool arrêté sans capacité)

## 3. Onglet Agents (`/agents`)

- [ ] 3.1 Dans `frontend/src/pages/AgentsPage.tsx`, afficher pour un pool actif l'en-tête « X/Y workers actifs » et le mode de délégation, le message « N workers disponibles, en attente d'un change prêt » lorsqu'il n'y a aucun worker, la ligne « N workers disponibles » sous le tableau lorsqu'il reste des workers libres, et l'état vide « aucun agent actif » uniquement si `is_running` est faux ; ajouter les clés `fr`/`en` dans `agents.json`
- [ ] 3.2 Étendre `frontend/src/pages/AgentsPage.test.tsx` avec les scénarios pool arrêté, pool actif sans worker, pool partiellement occupé ; vérifier que `npm test` passe pour ce fichier

## 4. Configuration > Agent Pool

- [ ] 4.1 Dans `AgentPoolTab` (`frontend/src/pages/ConfigurationPage.tsx`), remplacer le tableau vide par « N workers disponibles » pour un pool actif sans worker et ajouter la ligne « N workers disponibles » sous le tableau lorsqu'il reste des workers libres ; ajouter les clés `fr`/`en` dans `configuration.json`
- [ ] 4.2 Étendre `frontend/src/pages/ConfigurationPage.test.tsx` avec les scénarios pool sans worker et pool partiellement occupé ; vérifier que `npm test` passe pour ce fichier

## 5. Vérification d'intégration

- [ ] 5.1 Lancer la suite de tests frontend et le type-check, puis démarrer un pool de taille 3 sur un workspace sans change prêt et vérifier que « 0/3 » apparaît sur le bouton Kanban, dans l'onglet Agents et dans Configuration > Agent Pool
