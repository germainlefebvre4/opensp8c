## 1. Backend Development

- [x] 1.1 Enrichir la réponse JSON dans `backend/internal/api/handlers/preferences.go` pour y inclure un champ `systemEnv` contenant les variables recommandées de l'hôte (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`) récupérées via `os.Getenv`.
- [x] 1.2 Mettre à jour ou ajouter un test d'API pour le endpoint `GET /api/preferences` pour s'assurer que les variables système whitelistées sont correctement exposées.

## 2. Frontend Development

- [x] 2.1 Ajouter le champ optionnel `systemEnv: Record<string, string>` dans l'interface `Preferences` de `frontend/src/lib/api.ts`.
- [x] 2.2 Ajouter les traductions requises dans `frontend/src/locales/fr/dialogs.json` et `frontend/src/locales/en/dialogs.json` pour afficher l'état d'héritage système ou de surcharge utilisateur.
- [x] 2.3 Modifier `frontend/src/components/AgentSettingsModal.tsx` pour implémenter l'Option A :
  - Utiliser la valeur de `systemEnv` comme placeholder dynamique si la valeur personnalisée de l'utilisateur n'est pas définie.
  - Afficher un indicateur discret sous le champ (un badge vert avec coche pour "Sera héritée du système" ou un badge bleu/orange avec crayon pour "Surcharge la valeur système").

## 3. Validation

- [x] 3.1 Tester manuellement l'affichage en lançant le serveur avec une variable d'environnement définie (par exemple `GOOGLE_CLOUD_PROJECT=test-proj`) et vérifier le bon affichage dynamique du placeholder et des statuts dans la modal de configuration.
