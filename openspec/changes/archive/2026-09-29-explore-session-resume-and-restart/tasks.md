# Tasks

## 1. Vérification préalable du comportement de `--resume`

- [x] 1.1 Lancer manuellement `claude --print --input-format stream-json --output-format stream-json --resume <uuid-inconnu>` dans le workspace et noter le code de sortie et le délai avant sortie ; consigner le résultat en commentaire de la fonction `startWithResumeFallback` (tâche 3.1) et ajuster le délai de 1 s de `design.md` D3 si nécessaire

## 2. Modèle de données : `claudeSessionId` des explorations

- [x] 2.1 Ajouter `ClaudeSessionId string \`json:"claudeSessionId,omitempty"\`` à `ExplorationRecord` dans `backend/internal/preferences/preferences.go` et une méthode `SetExplorationClaudeSession(id, claudeSessionID)` ; vérifier avec un test dans `preferences_test.go` (aller-retour JSON, champ omis quand vide, ghost existant sans champ toujours chargé)
- [x] 2.2 Exposer `ClaudeSessionID` sur `session.Session` (lecture seule) et le renseigner dans `Start` et `StartAnonymous` ; vérifier avec un test de `manager_test.go`

## 3. Backend : continuité de session des explorations

- [x] 3.1 Ajouter à `Subprocess` un canal `exited` fermé par un unique goroutine appelant `Wait` (adapter `Wait` pour ne plus l'appeler deux fois) et implémenter `startWithResumeFallback` : démarrage en `--resume`, sortie précoce détectée dans la fenêtre d'attente, relance sans `--resume` avec nouvel UUID, retour d'un indicateur `contextLost` ; vérifier avec des tests dans `subprocess_test.go` (processus factice qui sort immédiatement vs qui reste actif)
- [x] 3.2 Faire utiliser `startWithResumeFallback` à `Manager.Start` en remplacement du fallback actuel ; vérifier que les tests existants de `manager_start_test.go` passent et ajouter un cas « `--resume` sort immédiatement → relance sans resume, nouvel id persisté »
- [x] 3.3 Faire lire à `StartAnonymous` le `claudeSessionId` de l'enregistrement (`GetExploration`) : `--resume` si présent, sinon UUID nouveau avec `--session-id` ; positionner `contextLost` pour les cas de l'exigence « Signalement d'un démarrage sans continuité de contexte » (échec de `--resume`, ghost sans id, agents sans support de reprise) ; vérifier avec des tests couvrant première ouverture, reprise réussie, ghost sans id et agent Antigravity
- [x] 3.4 Persister l'id dans `createGhostRecord` (lecture de `sess.ClaudeSessionID()`) et mettre à jour l'enregistrement quand `startWithResumeFallback` a changé l'id ; vérifier avec un test de `explore_test.go` qui relit `preferences.json`
- [x] 3.5 Dans `serveWS`, émettre `{"type":"session_restarted"}` avant le rejeu quand la session porte `contextLost` (une fois par connexion, jamais lors d'un rattachement à une session vivante) et `{"type":"replay_done"}` après le rejeu du snapshot ; vérifier avec des tests de handler dans `explore_test.go` (reprise réussie sans événement, échec de resume avec événement, rattachement sans événement, `replay_done` toujours présent)
- [x] 3.6 Rendre l'arrêt idempotent par identité : `Manager.Stop`/`StopAnonymous` (et le `onExpire` de `serveWS`) ne retirent l'entrée de la map que si elle est toujours la session attendue ; vérifier avec un test qui démarre une session, la remplace sous la même clé et vérifie que l'`onExpire` de l'ancienne ne stoppe pas la nouvelle
- [x] 3.7 Confirmer par test que `runPromoteFF` ne transmet aucun `claudeSessionId` de l'exploration (cas déjà couvert par le comportement actuel) ; ajouter l'assertion dans `explore_test.go`

## 4. Frontend : reprise sans réamorçage et sans doublons

- [x] 4.1 Supprimer l'envoi du message `[Reprise de session]` à `ws.onopen` dans `useAnonymousExploreSession.ts` ; le déplacer dans le traitement de `session_restarted` (une seule fois par connexion, texte demandant de poursuivre sans résumer, mêmes règles de troncature de `getStoredContext`) ; vérifier par un test du hook (ouverture sans événement : aucun `send` ; avec `session_restarted` : un seul `send`)
- [x] 4.2 Implémenter le dédoublonnage du rejeu (design D4) dans `exploreChat.ts` sous forme de fonctions pures (accumulation du rejeu dans une liste temporaire, application à `replay_done` avec la règle « messages suivant le dernier assistant stocké ») ; vérifier dans `exploreChat.test.ts` : localStorage vide, dernier message retrouvé, message manquant dans la fenêtre, tour terminé panneau fermé
- [x] 4.3 Brancher le rejeu dédoublonné dans `useAnonymousExploreSession.ts` et l'application directe dans `useExploreSession.ts` ; vérifier par un test de hook qu'une réattache à une session vivante n'affiche chaque réponse qu'une fois
- [x] 4.4 Ne plus dépendre de `getStoredContext` à l'ouverture tout en le conservant exporté pour `KanbanPage` ; vérifier que `KanbanPage.tsx` compile et que le test existant de promotion passe

## 5. Frontend : relance de l'agent

- [x] 5.1 Étendre `Message` avec le rôle `'system'`, l'exclure de la sérialisation localStorage et de `getStoredContext`, et rendre la ligne discrète dans `ExplorePanel.tsx` et `ExploreAnonymousPanel.tsx` ; vérifier par tests unitaires (`exploreChat.test.ts`) et par un rendu de composant
- [x] 5.2 Ajouter la détection de silence (design D6) : `lastInboundAt`, intervalle de 1 s actif seulement pendant `waiting`, seuils 60 s / 180 s (appel d'outil sans résultat), réinitialisation à `replay_done` ; exposer `stalled` ; vérifier avec des tests à horloge simulée couvrant les quatre scénarios de l'exigence « Détection d'une session d'exploration bloquée »
- [x] 5.3 Ajouter `restart()` à `useExploreSession.ts` (fermeture WS, `DELETE` de session, `connect`) et à `useAnonymousExploreSession.ts` (fermeture WS, `DELETE`, `POST` avec `resumeGhostId`, `connectWS`) ; ajouter la ligne système « Agent relancé » après un redémarrage réussi et un avertissement en cas d'échec ; vérifier par des tests de hook avec API et WebSocket simulés
- [x] 5.4 Afficher le bouton « Relancer l'agent » dans les deux panneaux quand `stalled` et `agentInfo.id === 'claude'`, sans masquer la saisie ; vérifier par tests de composant (visible/masqué, autre agent, clic appelle `restart`)
- [x] 5.5 Ajouter les libellés fr et en dans `frontend/src/locales/fr/explore.json` et `en/explore.json` (bouton, message système, avertissement d'échec) ; vérifier que le test d'i18n existant sur les clés manquantes passe

## 6. Intégration et documentation

- [x] 6.1 Vérifier en réel avec un agent Claude : ouvrir 3 fois de suite une exploration existante sans qu'aucun message d'assistant n'apparaisse ; laisser expirer le sous-processus (ou le tuer) puis rouvrir et vérifier la reprise sans réinjection ; simuler un blocage et vérifier que le bouton apparaît au bout de 60 s et relance la conversation avec « Agent relancé »
- [x] 6.2 Lancer `go test ./...` dans `backend` et les tests du frontend, et corriger les régressions éventuelles
- [x] 6.3 Mettre à jour la documentation existante qui décrit la reprise de session des explorations (recherche de « Reprise de session » et `explore-session-resume` dans `docs/` et le README) ; vérifier avec un `grep` que plus aucune page ne décrit l'injection systématique du transcript
