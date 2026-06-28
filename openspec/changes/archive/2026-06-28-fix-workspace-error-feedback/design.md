## Context

Le client HTTP Axios lève une `AxiosError` quand le serveur répond avec un statut 4xx/5xx. Par défaut, `AxiosError.message` contient un message générique (`"Request failed with status code 422"`). Le corps de la réponse — qui contient le message descriptif envoyé par le backend Go via `http.Error()` — est accessible dans `AxiosError.response.data`.

Actuellement, aucun intercepteur n'est configuré sur l'instance Axios partagée (`frontend/src/lib/api.ts`). Chaque composant gère les erreurs lui-même avec `err.message`, ce qui perd l'information du serveur.

## Goals / Non-Goals

**Goals:**
- Surfacer le message d'erreur du corps de la réponse HTTP dans tous les flux existants sans modifier les composants consommateurs
- Couvrir les deux formats que le backend peut renvoyer : texte brut (`http.Error`) et JSON (`{ "error": "..." }`)

**Non-Goals:**
- Modifier le backend ou le format des erreurs serveur
- Ajouter un système de notification toast global
- Internationalisation des messages d'erreur

## Decisions

### Intercepteur Axios plutôt que wrapper par composant

**Décision** : Ajouter un intercepteur `response` sur l'instance `api` partagée dans `api.ts`.

**Pourquoi** : Un seul point de changement couvre tous les composants actuels et futurs. L'alternative — patcher chaque `catch` — est fragile et crée de la dette dès l'ajout de nouveaux handlers.

**Alternative écartée** : Helper `parseAxiosError(err)` importé dans chaque composant. Trop de surface à maintenir, oubli possible dans les nouveaux composants.

### Extraction du message — priorité texte brut > JSON > fallback

**Décision** : L'intercepteur tente dans l'ordre :
1. Corps texte brut (`typeof data === 'string'`) → `.trim()`
2. Champ JSON `data.error` ou `data.message`
3. Fallback sur `err.message` (comportement actuel, préservé)

**Pourquoi** : Le backend Go utilise `http.Error()` qui envoie du texte brut. Gérer aussi le JSON prépare à une évolution future sans nouveau changement.

### Retourner un `Error` standard

**Décision** : L'intercepteur relève un `Error` standard (pas un `AxiosError`) avec le message extrait.

**Pourquoi** : Les composants font déjà `err instanceof Error ? err.message : '...'`. En retournant un `Error` standard, tous les catch existants fonctionnent sans modification.

## Risks / Trade-offs

- **Perte du statut HTTP dans les composants** → Les composants qui auraient besoin du code de statut ne pourront plus le lire via `err.response.status`. Mitigation : acceptable pour les cas actuels ; si un composant a besoin du statut, il peut utiliser `axios.isAxiosError(err)` avant le reject de l'intercepteur.
- **Corps de réponse non-UTF8** → Si le serveur renvoie du binaire sur une erreur, `trim()` ne provoque pas d'erreur mais le message sera illisible. Cas non applicable ici (Go envoie toujours du texte).
