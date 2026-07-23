## 1. Infrastructure & Configuration

- [x] 1.1 Spécialiser `BuildSubprocessArgs` dans `backend/internal/agents/agents.go` pour générer les arguments spécifiques à Gemini.
- [x] 1.2 Adapter la gestion de `--session-id` et de `--resume` dans `backend/internal/session/subprocess.go` pour ne pas bloquer l'agent Gemini.
- [x] 1.3 Documenter ou s'assurer de la propagation sans altération des variables d'environnement (`GOOGLE_CLOUD_PROJECT`).

## 2. Adaptation du Protocole de Streaming

- [x] 2.1 Mettre en place un mécanisme d'adaptation pour convertir l'entrée standard (messages utilisateur du frontend vers l'entrée textuelle/JSON attendue par le CLI Gemini).
- [x] 2.2 Implémenter l'adaptation du flux de sortie (parser le NDJSON émis par Gemini CLI pour reconstituer des chunks de texte éligibles pour le format Claude-like attendu par le frontend).
- [x] 2.3 Écrire des tests unitaires ou d'intégration locaux dans le backend pour valider le lancement robuste du sous-processus de l'agent Gemini.
