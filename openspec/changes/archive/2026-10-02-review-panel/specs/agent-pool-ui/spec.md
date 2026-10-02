# Spec Delta

## REMOVED Requirements

### Requirement: Panneau interactif de Review HITL (Human-In-The-Loop)
**Reason**: Ce requirement mêlait trois responsabilités dans un panneau dédié ouvert « au clic sur la carte » (affichage des fichiers et du diff, bouton d'approbation, champ de feedback), jamais implémenté. L'affichage devient un onglet « Revue » du `DetailPanel`, et l'approbation et la demande de correction vivent dans l'onglet Actions avec leurs propres dialogues. La correction ne repose plus sur un feedback injecté dans l'invite système du worker mais sur une tâche ajoutée à `tasks.md`.
**Migration**: Pour l'affichage des fichiers modifiés et du diff, voir `change-review-panel`. Pour « Approuver et Fusionner » et « Demander des Corrections », voir `change-review-actions`.
