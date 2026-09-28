# Spec Delta

## REMOVED Requirements

### Requirement: Affichage adaptatif de la configuration de l'agent
**Reason**: Ce comportement décrivait `AgentSettingsModal`, qui est retirée de l'interface (icône engrenage supprimée du sélecteur d'agent dans la sidebar). La configuration des variables recommandées et système est désormais spécifiée sous la capability `platform-configuration`.
**Migration**: Voir `platform-configuration` > "Affichage adaptatif des variables recommandées" (page Configuration > CLI).

### Requirement: Réglage global du mode question native
**Reason**: Cette bascule vivait dans `AgentSettingsModal`, retirée au profit de la page Configuration globale, indépendante du workspace.
**Migration**: Voir `platform-configuration` > "Réglage du mode question native" (page Configuration > CLI).
