## Context

Le serveur backend `opensp8c` lance les processus fils d'agent CLI en combinant les variables d'environnement système de sa machine hôte (`os.Environ()`) et les variables personnalisées définies par l'utilisateur (`preferences.json`). 
Cependant, l'interface web de configuration n'affiche aucune information sur l'état des variables d'environnement système, laissant l'utilisateur dans l'incertitude quant à savoir si des variables importantes comme `GOOGLE_CLOUD_PROJECT` sont déjà valorisées au niveau du système d'exploitation sous-jacent.

## Goals / Non-Goals

**Goals:**
- Exposer de manière dynamique et sécurisée les variables d'environnement hôtes pertinentes pour l'agent au client Web.
- Fournir une UI ergonomique adaptative (Option A) distinguant visuellement les variables héritées du système de celles explicitement surchargées par l'utilisateur.

**Non-Goals:**
- Exposer la totalité des variables système (`os.Environ()`) pour éviter toute fuite accidentelle d'informations sensibles ou de secrets locaux de l'hôte.
- Enregistrer les variables système de l'hôte dans `preferences.json` (qui doit uniquement contenir les surcharges volontaires de l'utilisateur).

## Decisions

### Décision 1 : Enrichissement de l'endpoint GET /api/preferences plutôt qu'un endpoint séparé
- **Option A (Endpoint séparé /api/system-env) :** (Rejeté : Complexifie le frontend en nécessitant plusieurs hooks React Query et augmente le nombre de requêtes réseau lors de l'ouverture de la modal).
- **Option B (Enrichissement de GET /api/preferences - Sélectionnée) :** Ajouter un dictionnaire `systemEnv` dans la structure de réponse JSON des préférences. Cela simplifie la gestion d'état dans le frontend et garantit que toutes les informations de configuration nécessaires à la modal sont chargées de manière atomique.

### Décision 2 : Whitelisting strict des variables système à exposer
- **Option A (Exposition complète) :** (Rejeté : Présente des risques majeurs de sécurité si des clés privées, jetons d'accès SSH, ou variables système confidentielles sont exposés).
- **Option B (Whitelist stricte - Sélectionnée) :** Le backend n'exposera explicitement que les variables recommandées de l'agent : `GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, et `GEMINI_SANDBOX`.

### Décision 3 : Choix de l'Option A d'UI (Placeholders + Messages de statut)
- **Option A (Placeholders & Indicateurs contextuels - Sélectionnée) :** Si la surcharge utilisateur est vide, l'input utilise la valeur système en placeholder et affiche sous l'input : `"✔ Sera héritée de l'environnement système"`. Si l'utilisateur saisit sa surcharge, l'input affiche la valeur saisie et le message passe à : `"✎ Surcharge la valeur système"`. Cela offre une excellente clarté et un feedback immédiat.
- **Option B (Badges statiques côte à côte) :** (Rejeté : Alourdit l'interface visuellement et prend trop de place dans une modal compacte).

## Risks / Trade-offs

- **[Risque] Valeurs système périmées si le serveur est en cours d'exécution**
  - *Mitigation :* Les valeurs de `systemEnv` sont résolues de manière dynamique à la volée via `os.Getenv` à chaque requête de lecture de préférences, garantissant un reflet exact de l'environnement du serveur.
