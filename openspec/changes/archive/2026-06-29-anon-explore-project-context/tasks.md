## 1. Backend — Retrait du message d'amorce

- [x] 1.1 Dans `backend/internal/session/manager.go`, supprimer le bloc `initPayload` / `initMsg` / `proc.Write` dans `StartAnonymous` (lignes ~181-190)

## 2. Backend — Interception du premier message

- [x] 2.1 Ajouter le paramètre `anonymous bool` à la signature de `serveWS` dans `backend/internal/api/handlers/explore.go`
- [x] 2.2 Dans `serveWS`, déclarer `firstSent := false` et, si `anonymous && !firstSent`, appeler `prependExploreSkill(msg)` puis passer `firstSent = true`
- [x] 2.3 Implémenter `prependExploreSkill(msg []byte) []byte` : parser le JSON, préfixer le champ `content` avec `/opsx:explore `, re-sérialiser ; retourner `msg` tel quel si le parse échoue
- [x] 2.4 Mettre à jour les deux appels à `serveWS` : `false` dans `HandleWS`, `true` dans `HandleAnonymousWS`

## 3. Frontend — Message statique

- [x] 3.1 Dans `ExploreAnonymousBottomPanel` (ou le composant qui affiche les messages), initialiser l'état `messages` avec un message statique d'invitation plutôt qu'un tableau vide
- [x] 3.2 Supprimer l'état `waiting` initial (aujourd'hui déclenché en attente du greeting backend)
