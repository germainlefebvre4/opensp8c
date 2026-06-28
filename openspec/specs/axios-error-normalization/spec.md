# Spec: Axios Error Normalization

## Purpose

Normalisation des erreurs HTTP émises par le client Axios partagé (`api`), afin de surfacer le message d'erreur du serveur en lieu et place du message générique Axios, sans modifier les composants consommateurs.

## Requirements

### Requirement: Erreurs HTTP surfacées comme message lisible
Le client HTTP SHALL extraire le corps de la réponse serveur comme message d'erreur lorsqu'une requête échoue avec un statut 4xx ou 5xx, à la place du message générique Axios.

#### Scenario: Erreur avec corps texte brut
- **WHEN** le serveur répond avec un statut 4xx et un corps texte brut (ex. `"directory does not contain an openspec/ folder\n"`)
- **THEN** l'erreur levée côté client a pour message le texte du corps, sans le `\n` final

#### Scenario: Erreur avec corps JSON contenant un champ `error`
- **WHEN** le serveur répond avec un statut 4xx et un corps JSON de la forme `{ "error": "..." }`
- **THEN** l'erreur levée côté client a pour message la valeur du champ `error`

#### Scenario: Erreur sans corps exploitable
- **WHEN** le serveur répond avec un statut 4xx et un corps vide ou non-parsable
- **THEN** l'erreur levée côté client conserve le message Axios générique (comportement de fallback)

#### Scenario: Requête réseau échouée (pas de réponse)
- **WHEN** la requête échoue sans réponse du serveur (timeout, réseau indisponible)
- **THEN** l'erreur levée est transmise sans modification

### Requirement: Couverture globale sans modification des composants
Le mécanisme de normalisation des erreurs SHALL s'appliquer à toutes les requêtes émises via l'instance `api` partagée sans que les composants consommateurs aient à modifier leur gestion d'erreurs.

#### Scenario: Composant existant bénéficie du fix
- **WHEN** un composant catch une erreur avec `err instanceof Error ? err.message : 'Erreur inconnue'`
- **THEN** `err.message` contient le message du serveur, pas le message générique Axios
