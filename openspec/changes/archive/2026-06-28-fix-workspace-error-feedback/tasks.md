## 1. Intercepteur Axios

- [x] 1.1 Ajouter un intercepteur `response` sur l'instance `api` dans `frontend/src/lib/api.ts` qui extrait le corps texte brut ou JSON (`error`/`message`) comme message d'erreur
- [x] 1.2 Vérifier que les requêtes sans réponse (erreurs réseau) sont transmises sans modification

## 2. Vérification manuelle

- [x] 2.1 Tester l'ajout d'un projet avec un chemin valide mais sans dossier `openspec/` — vérifier que le message affiché est `"directory does not contain an openspec/ folder"`
- [x] 2.2 Tester l'ajout d'un projet avec un chemin inexistant — vérifier que le message serveur est bien surfacé
