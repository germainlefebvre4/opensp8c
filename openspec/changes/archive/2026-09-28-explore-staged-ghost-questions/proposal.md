# Proposal

## Why

Lors des sessions d'exploration, l'agent pose fréquemment plusieurs questions de clarification simultanées pour cadrer le besoin (par exemple sous forme de marqueurs `ghost_question`). Actuellement, les questions multiples provoquent l'apparition de JSON brut dans le fil de conversation en raison d'un stripping incomplet dans le flux de streaming, et chaque nouvelle question rend la précédente inactive. De plus, répondre à une question déclenche un envoi WebSocket immédiat, empêchant l'utilisateur de préparer ses réponses à son rythme, de les corriger avant envoi, et d'y adjoindre un prompt libre dans un envoi unique et maîtrisé.

Cette évolution permet de détecter proprement toutes les questions de clarification sans fuite de JSON brut, d'activer la saisie locale et différée des réponses (« staging ») avec encarts temporaires dans le fil de conversation, et de laisser l'utilisateur seul maître de l'envoi consolidé.

## What Changes

- **Backend - Détection et stripping multi-questions** : Détecter et diffuser tous les marqueurs `ghost_question` émis dans un même tour ou flux, et nettoyer intégralement les marqueurs du streaming texte (`content_block_delta` et messages consolidés) pour éliminer les fuites de JSON brut dans le chat.
- **Frontend - Support de questions concurrentes** : Permettre à plusieurs cartes de questions d'être actives simultanément dans le fil de discussion (fin du marquage arbitraire en `superseded` lors de l'arrivée d'une nouvelle question).
- **Frontend - Saisie et staging des réponses par question** : Intégrer un champ de saisie direct sur chaque carte de question pour préparer une réponse localement sans envoi immédiat vers le subprocess.
- **Frontend - Encarts de réponses préparées dans le fil** : Afficher des encarts temporaires dans la fenêtre de conversation récapitulant les réponses préparées, avec possibilité d'édition et de suppression avant envoi.
- **Frontend - Envoi consolidé et persistance des questions non répondues** : Permettre l'envoi du message consolidé (réponses préparées + prompt libre optionnel) via le bouton d'envoi principal du chat. Dès l'envoi, les encarts temporaires disparaissent, les questions traitées passent au statut répondu, et les questions laissées sans réponse restent ouvertes pour le tour suivant.

## Capabilities

### Modified Capabilities
- `explore-structured-questions`: Mise à jour pour supporter la détection multiple et le stripping strict sans fuite, la coexistence de plusieurs questions actives, la saisie locale préparée (staged) avec encarts dans le fil de discussion, et l'envoi consolidé laissant les questions non répondues ouvertes.

## Impact

- **Backend** : `internal/api/handlers/explore.go`, `internal/session/manager.go`, et tests associés.
- **Frontend** : `src/hooks/exploreChat.ts`, `src/components/QuestionCard.tsx`, `src/hooks/useExploreSession.ts`, `src/hooks/useAnonymousExploreSession.ts`, `src/components/ExplorePanel.tsx`, `src/components/ExploreAnonymousPanel.tsx`, et tests unitaires associés.
- **Contrat WebSocket** : Compatibilité préservée ; le format des événements `ghost_question` émis reste standard, et le message envoyé au subprocess est un message utilisateur enrichi lisible par tout LLM.
