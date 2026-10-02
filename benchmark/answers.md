Réponses aux questions de clarification (à donner telles quelles, une fois la question posée) :

- **Qu'appelle-t-on un « run » ?** Uniquement les runs du pool d'agents : les fichiers `.jsonl` du dossier `pool/` des logs de conversation du change (`<conversations>/<workspaceID>/<change>/pool/`). Les runs de chat, d'explore et de ff ne comptent pas. Un run non terminé (sans marqueur de fin) compte dans `runs_count`.
- **Comment se calcule la durée d'un run ?** Du marqueur `pool_run_start` au marqueur `pool_run_end` du journal du run (champ `ts` de chaque ligne, RFC 3339). Un run sans marqueur de fin ajoute 0 seconde. `total_duration_seconds` est la somme sur tous les runs, un nombre (les fractions de seconde sont permises).
- **Comment se calcule `progress_percent` ?** Entier, `tasks_done * 100 / tasks_total` arrondi à l'inférieur ; 0 s'il n'y a aucune tâche.
- **Un change sans aucun run ?** `runs_count` vaut 0 et `total_duration_seconds` vaut 0, avec une réponse 200.
- **Format des tâches ?** Celui de `tasks.md` : les lignes `- [ ]` et `- [x]`, comme le reste de la plateforme.
