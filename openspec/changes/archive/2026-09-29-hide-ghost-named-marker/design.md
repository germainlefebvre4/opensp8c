# Design

## Context

Le backend relaye chaque message du subprocess agent au WebSocket via une goroutine de sortie (`explore.go`). `detectGhostQuestion` retire `ghost_question` par regex sur les valeurs string d'un message JSON décodé ; rien ne retire `ghost_named` (il est seulement extrait pour renommer le ghost et émettre l'événement WS `ghost_named`). Avec le streaming, le marker arrive en `content_block_delta` d'un token chacun (`{"`, `event"`, …), puis à nouveau en un bloc dans le message `assistant` consolidé. Le front concatène les deltas dans `mergeAssistantText`, dont le nettoyage résiduel ne connaît que `ghost_question` et ne voit qu'un delta à la fois. Voir proposal.md pour la motivation.

## Goals / Non-Goals

**Goals:**
- Aucun fragment du marker `ghost_named` visible, ni en live ni en relecture.
- Un nettoyage réutilisable par d'autres markers `ghost_*`.
- Une notice de nommage discrète, persistée, sans doublon.

**Non-Goals:**
- Migrer `ghost_question` / `ghost_draft` vers le nouveau mécanisme.
- Modifier le protocole WS ou le prompt système de l'agent.
- Animer le header ou afficher un toast.

## Decisions

**1. Filtre à état côté backend, par flux de texte.** Un filtre alimenté delta par delta retient le texte tant qu'il peut être le début d'un marker `{"event":"ghost_…` en début de ligne. Il le supprime dès que l'objet JSON est complet (accolade fermante hors chaîne) et restitue le texte retenu s'il s'invalide. Il est vidé à `content_block_stop` / fin de tour pour ne jamais perdre de texte. Le message `assistant` consolidé continue d'être nettoyé par regex sur les valeurs décodées, étendue à `ghost_named`. *Alternative écartée* : bufferiser toute la première réponse — casse le streaming.

**2. Filtre pur, sans état, côté frontend sur le contenu accumulé.** `mergeAssistantText` ajoute le delta brut puis applique `stripGhostMarkers(content)` : suppression des markers complets et suppression d'un suffixe partiel qui est un préfixe de marker. Comme le contenu accumulé est réévalué à chaque delta, aucune mémoire n'est nécessaire, (la timeline de `DetailPanel` vient des logs d’activité, hors périmètre). C'est un filet de sécurité ; le backend reste la source de vérité.

**3. Rôle `notice` dans `Message`.** `Message.role` accepte `'notice'` (contenu = nom final). Les panels le rendent en ligne compacte (icône + libellé i18n `explore:namedNotice`) sans indicateur de rôle. Les messages `notice` sont sauvegardés avec l'historique existant (`saveMessages`). *Alternative écartée* : ligne éphémère dans le header — rien dans l'historique.

**4. Anti-doublon de la notice.** Elle n'est insérée que sur l'événement WS `ghost_named` et seulement si aucune notice n'existe déjà dans les messages restaurés ; une nouvelle valeur remplace la notice existante au lieu d'en ajouter une.

## Risks / Trade-offs

- [Un `{` légitime en début de réponse est retenu un court instant] → le filtre ne retient que tant que le texte reste un préfixe valide du marker, puis restitue tout dans l'ordre.
- [Le marker n'est jamais complété (agent tronqué)] → vidage à la fin de bloc ; le texte retenu est restitué.
- [Formats d'agents différents (Gemini traduit)] → le filtre opère sur `delta.text` après traduction et sur les valeurs string décodées, comme l'extraction existante ; à couvrir par des tests.
- [Deux implémentations (Go, TS)] → mêmes cas de test de part et d'autre.
