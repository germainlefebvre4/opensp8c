# Design

## Context

Voir `proposal.md` (Why) pour la motivation. Contraintes techniques actuelles constatées dans le code :

- Chaque session (nommée ou anonyme) fait tourner **un seul subprocess long-lived** par conversation (`session/manager.go`, `Session`/`Manager`), piloté en `stream-json` bidirectionnel. Rien n'est réinitialisé entre les messages d'un même fil.
- Le seul mécanisme "structuré" existant est une **convention de marqueur texte** : l'agent écrit littéralement `{"event":"ghost_named","name":"..."}` en tout début de première réponse, détecté côté backend par `ExtractGhostNamed` (JSON strict puis repli en recherche de sous-chaîne, y compris à travers le format `content_block_delta` traduit par le pont Gemini). Aucun vrai bloc `tool_use`/`tool_result` n'est traité nulle part dans le code actuel (`grep` à vide sur backend et frontend).
- Le pont de traduction Gemini (`session/subprocess.go`, `translateGeminiLine`) ne connaît que les types `message`, `init`, `result` ; tout autre type de ligne JSON traverse tel quel (`return line`).
- L'interdiction actuelle d'`AskUserQuestion` est un texte figé (`baseSystemPrompt` dans `subprocess.go`) déjà repris littéralement dans le spec `explore-session`.
- `AgentSettingsModal.tsx` + `useAgentPreferences.ts` gèrent déjà un objet de préférences persisté côté backend (`preferences.json`), aujourd'hui limité à `defaultAgent` et un dictionnaire `env`.
- Le message d'accueil affiché avant le premier message utilisateur (`STATIC_GREETING`) est un texte frontend statique, jamais produit par l'agent — le vrai subprocess ne démarre qu'à l'envoi du premier message.

## Goals / Non-Goals

**Goals:**
- Un mécanisme de question structurée qui fonctionne identiquement sur tous les agents CLI supportés, sans coupler le rendu à un protocole d'outil spécifique.
- Un chemin optionnel, réservé à Claude, qui exploite le vrai outil `AskUserQuestion` quand l'utilisateur l'active.
- Une reprise de session robuste qui ne dépend jamais de l'état interne d'un sous-processus détruit.

**Non-Goals:**
- Pas d'équivalent natif pour Codex, Gemini, Copilot ou Antigravity, même s'ils exposent un jour un outil de question interactif — hors périmètre de ce change.
- Pas de réglage par session/conversation pour le mode question native : uniquement un réglage global, conformément au choix fait en exploration.
- Pas de modification des fichiers de skill vendorisés par agent (`.claude/skills`, `.gemini/skills`, `.agents/skills`) : toute instruction de cadrage/marqueur passe uniquement par `--append-system-prompt`.
- Pas de widget dédié pour les questions à sélection multiple (`multiSelect`) côté mode natif : une question native multi-sélection retombe sur la redirection "Autre réponse" vers le champ principal plutôt que sur des cases à cocher.

## Decisions

### 1. Marqueur `ghost_question`, détection identique à `ghost_named`
Réutilise le pattern déjà éprouvé (`ExtractGhostNamed`) plutôt que d'introduire un nouveau protocole : une fonction `ExtractGhostQuestion` symétrique, tolérante (JSON strict, puis repli sur recherche de sous-chaîne, y compris via le format traduit de Gemini). Alternative écartée : passer par un vrai appel d'outil pour tous les agents — rejeté car cela recouplerait le mécanisme universel à un protocole `tool_use` que seul Claude expose de façon exploitable ici.

### 2. État "question en attente" porté par la `Session`, pas persisté séparément
`Session` (manager.go) gagne un état de question en attente, écrit à la détection du marqueur (ou du `tool_use` natif) et effacé dès l'envoi du message utilisateur suivant, protégé par le mutex existant (`msgMu`). À la reprise, cet état est reconstruit en scannant la fin du buffer de messages (in-memory pour une session active, log JSONL rejoué pour une session interrompue / historique localStorage pour l'anonyme) à la recherche d'une ligne marqueur sans réponse utilisateur postérieure — plutôt que d'ajouter un nouveau champ persisté dans `preferences.json`. Alternative écartée : un booléen dédié persisté à côté de `claudeSessionId` — rejeté pour éviter un état à garder synchronisé en plus du log déjà source de vérité.

### 3. Prompt système étendu, symétrique nommé/anonyme
`baseSystemPrompt` (subprocess.go) et `anonSystemPrompt` (manager.go) sont étendus avec les mêmes instructions de cadrage progressif + convention `ghost_question`. Pour les sessions nommées, `Manager.Start` passe aujourd'hui `""` comme `extraSystemPrompt` — ce change introduit un prompt partagé non vide pour aligner le comportement nommé/anonyme, sans toucher au message auto-injecté `/opsx:explore <changeName>` qui reste inchangé.

### 4. Mode natif Claude : parsing `tool_use` isolé, activé conditionnellement
Quand l'agent résolu est Claude et que la préférence globale est active, le `baseSystemPrompt` construit pour cette session omet la clause d'interdiction. Le goroutine de fan-out stdout (`startFanOut`) gagne une étape de reconnaissance des blocs `content_block_start`/`content_block_stop` de type `tool_use` nommés `AskUserQuestion` : ils sont extraits du flux (jamais transmis tels quels comme texte), transformés en un événement `native_question` dédié, et la réponse utilisateur est réinjectée comme un bloc `tool_result` sur le stdin du subprocess (référencé par le `tool_use_id` d'origine), via le même chemin d'écriture que les messages texte normaux (`Subprocess.Write`).

### 5. Un seul composant de carte, deux sources d'événements
Le frontend introduit un composant `QuestionCard` unique consommé aussi bien par l'événement `ghost_question` (texte + options éventuellement suggérées par l'agent dans le marqueur) que par l'événement `native_question` (structure `questions`/`options` du vrai outil). Le style (carte détachée, collapse en résumé, redirection "Autre réponse" vers le `ref` du champ de saisie principal) est identique dans les deux cas — seule la source de données diffère. Alternative écartée : deux composants séparés — rejeté, cela dupliquerait le style validé en exploration sans bénéfice.

### 6. Réglage natif dans `preferences.json`, hors du dictionnaire `env`
Le nouveau réglage est un champ booléen de premier niveau (ex. `nativeQuestionMode`) distinct du dictionnaire `env` existant, car il ne s'agit pas d'une variable d'environnement injectée au subprocess mais d'un comportement de parsing backend. Exposé par les mêmes endpoints `GET`/`PATCH /api/preferences`.

## Risks / Trade-offs

- **[Risk]** Un texte halluciné par l'agent pourrait ressembler au marqueur `ghost_question` sans intention réelle de poser une question → **Mitigation**: détection stricte sur la clé JSON `"event":"ghost_question"` (comme pour `ghost_named`) ; le pire cas dégénère en une carte affichée à tort, jamais en erreur ou crash.
- **[Risk]** Le pont Gemini transmet aujourd'hui tel quel tout type d'événement inconnu ; un futur outil interactif Gemini pourrait un jour émettre un bloc similaire à `tool_use`/`AskUserQuestion` → **Mitigation**: le repli silencieux (requirement dédié dans `explore-native-question-mode`) filtre explicitement tout bloc `tool_use` nommé `AskUserQuestion` avant transmission, quel que soit l'agent, indépendamment de l'activation du mode natif.
- **[Risk]** Reconstruire l'état "question en attente" en scannant la fin du buffer/log à la reprise est plus fragile qu'un état persisté explicite si le format du marqueur venait à changer → **Mitigation**: le marqueur est toujours émis seul sur sa propre ligne, comme `ghost_named` déjà en production ; le scan cherche une ligne dédiée, pas un sous-texte noyé dans une réponse plus longue.
- **[Risk]** Le round-trip `tool_use`/`tool_result` avec Claude en mode `--print --input-format stream-json` n'a aucun précédent dans ce code (`grep` à vide confirmé) — surface d'erreur nouvelle → **Mitigation**: chemin entièrement derrière un réglage désactivé par défaut, avec repli garanti sur le marqueur texte déjà universel en cas d'échec de parsing ou de non-réponse du protocole.

## Migration Plan

Changement additif : le nouveau champ de préférence est absent par défaut des `preferences.json` existants (traité comme désactivé) ; le nouveau prompt système ne modifie que le comportement futur des sessions, sans migration de données. Rollback possible en revenant au prompt système précédent et en masquant le réglage, sans invalider d'état persisté existant.

## Open Questions

- Formulation exacte (fr/en) du message de suivi injecté pour faire reposer une question à la reprise — détail de contenu, sans impact sur les specs ou la répartition des tâches ; à trancher lors de l'implémentation.
