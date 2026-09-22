# Tasks

## 1. Marqueur `ghost_question` et état "question en attente" (backend)

- [x] 1.1 Ajouter `ExtractGhostQuestion` dans `backend/internal/session/manager.go`, symétrique à `ExtractGhostNamed` (JSON strict puis repli sous-chaîne, y compris format traduit Gemini) ; vérifier avec un test unitaire couvrant les mêmes cas que `ExtractGhostNamed` (JSON direct, `content_block_delta`, échappé)
- [x] 1.2 Ajouter un état de question en attente sur `Session` (manager.go), protégé par `msgMu`, positionné à la détection du marqueur et effacé à l'envoi du message utilisateur suivant ; vérifier par test unitaire que l'état bascule correctement dans les deux sens
- [x] 1.3 Diffuser un événement `ghost_question` distinct au frontend dès détection (à côté de l'événement `ghost_named` existant dans `api/handlers/explore.go`) ; vérifier via test d'intégration handler que l'événement WebSocket est bien émis avec le texte de la question

## 2. Prompt système : cadrage progressif et marqueur (backend)

- [x] 2.1 Étendre `baseSystemPrompt` (`session/subprocess.go`) et `anonSystemPrompt` (`session/manager.go`) avec les instructions de cadrage du premier tour et la convention `ghost_question` ; vérifier en relisant le prompt injecté dans les logs d'un subprocess de test
- [x] 2.2 Faire passer `Manager.Start` (sessions nommées) d'un `extraSystemPrompt` vide à ce même prompt partagé, sans toucher à l'auto-injection `/opsx:explore <changeName>` ; vérifier qu'une session nommée fraîchement créée reçoit bien le prompt étendu (test existant sur `Manager.Start` mis à jour)

## 3. Mode question native Claude (backend)

- [x] 3.1 Ajouter le champ booléen `nativeQuestionMode` à `preferences.json` (service de préférences), exposé par `GET`/`PATCH /api/preferences`, désactivé par défaut ; vérifier par test d'API que la valeur par défaut est `false` et qu'un `PATCH` la persiste
- [x] 3.2 Construire une variante du prompt système sans la clause d'interdiction `AskUserQuestion` quand l'agent résolu est Claude et que `nativeQuestionMode` est actif pour la session ; vérifier par test unitaire de résolution de prompt sur les 3 cas (Claude+actif, Claude+inactif, non-Claude+actif)
- [x] 3.3 Détecter les blocs `content_block_start`/`content_block_stop` de type `tool_use` nommés `AskUserQuestion` dans le fan-out stdout (`startFanOut`, manager.go), les extraire du flux (jamais transmis comme texte brut) et émettre un événement `native_question` dédié ; vérifier par test avec un flux stream-json simulé contenant un tel bloc
- [x] 3.4 Écrire le `tool_result` correspondant sur le stdin du subprocess Claude à réception de la réponse utilisateur à une question native, référencé par le `tool_use_id` d'origine ; vérifier par test d'intégration que le message stdin généré est bien formé
- [x] 3.5 Filtrer silencieusement tout bloc `tool_use` nommé `AskUserQuestion` qui traverserait le pont de traduction Gemini (`translateGeminiLine`) ou tout agent non-Claude, pour éviter un affichage de JSON brut ; vérifier par test que ce type de ligne ne produit aucune sortie exploitable par le frontend

## 4. Reprise de session avec question en attente (backend)

- [x] 4.1 Implémenter la reconstruction de l'état "question en attente" par scan de la fin du buffer/log de session (named: log JSONL ; anonyme: buffer in-memory) à la recherche d'un marqueur sans réponse utilisateur postérieure ; vérifier par test unitaire sur un buffer de messages construit à la main
- [x] 4.2 Injecter un message de suivi automatique demandant à l'agent de reformuler sa question, au redémarrage d'un subprocess (named via `--resume`, anonyme via replay de contexte) quand l'état reconstruit indique une question en attente ; vérifier par test d'intégration sur `Manager.Start` avec un `claudeSessionId` existant et un log se terminant sur un marqueur non répondu

## 5. Rendu frontend de la carte de question

- [x] 5.1 Créer le composant `QuestionCard` (carte détachée : fond blanc, bordure, ombre légère) consommant indifféremment un événement `ghost_question` ou `native_question` ; vérifier visuellement dans le panneau d'exploration avec un événement simulé de chaque type
- [x] 5.2 Implémenter l'état "répondu" : collapse de la carte en ligne "-> Votre réponse : ..." après envoi d'une réponse ; vérifier manuellement qu'une carte répondue reste lisible en remontant l'historique du fil
- [x] 5.3 Implémenter le comportement "Autre réponse..." : déplacement du focus vers le champ de saisie principal sans ouvrir de champ inline dans la carte ; vérifier manuellement le focus clavier après clic
- [x] 5.4 Garantir qu'une seule carte de question active (non répondue) n'est jamais affichée en simultané, même si l'agent enchaîne plusieurs marqueurs avant réponse ; vérifier avec un scénario de deux marqueurs consécutifs sans réponse intermédiaire
- [x] 5.5 Intégrer `QuestionCard` dans `useExploreSession.ts` et `useAnonymousExploreSession.ts` (parsing des événements `ghost_question`/`native_question`) et supprimer le `STATIC_GREETING` codé en dur ; vérifier qu'ouvrir une nouvelle exploration n'affiche plus de message d'accueil statique avant le premier message

## 6. Réglage "mode question native" (frontend)

- [x] 6.1 Ajouter la bascule "mode question native (Claude)" dans `AgentSettingsModal.tsx`, avec mention explicite qu'elle est sans effet sur les autres agents, branchée sur `useAgentPreferences.ts` ; vérifier manuellement que l'état de la bascule est chargé et sauvegardé via `PATCH /api/preferences`

## 7. Internationalisation

- [x] 7.1 Ajouter les nouvelles clés i18n (fr/en) pour la carte de question, l'état répondu, le lien "Autre réponse...", et la bascule de réglage, dans `locales/fr/explore.json`, `locales/en/explore.json` et les fichiers `dialogs`/`kanban` concernés ; vérifier qu'aucun texte nouveau n'est codé en dur en cohérence avec le change `i18n-full-coverage` en cours

## 8. Vérification de bout en bout

- [ ] 8.1 Scénario complet sur un agent non-Claude (ex. Codex ou Gemini) : ouverture d'exploration, question de cadrage posée via marqueur, réponse, carte réduite en résumé ; vérifier manuellement de bout en bout
- [ ] 8.2 Scénario complet sur Claude avec le mode natif activé : question native affichée, réponse, `tool_result` renvoyé, conversation poursuivie normalement ; vérifier manuellement de bout en bout
- [x] 8.3 Scénario de reprise : fermer l'onglet pendant qu'une question est en attente (marqueur ou native), rouvrir, vérifier que l'agent repose la question et que l'utilisateur peut y répondre normalement
