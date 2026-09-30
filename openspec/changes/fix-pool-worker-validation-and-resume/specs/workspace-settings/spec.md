# Spec Delta

## ADDED Requirements

### Requirement: Surcharge de la commande de validation dans Settings
Le sous-onglet Agent Pool de Settings SHALL permettre de surcharger, pour le workspace, la commande de validation exécutée par les workers du pool après chaque invocation de l'agent (`validationCommand`, texte libre). Lorsque le workspace ne définit pas de surcharge, le champ SHALL afficher la valeur héritée de Configuration, ou l'indication que la commande est auto-détectée si Configuration n'en définit pas. La surcharge SHALL pouvoir être réinitialisée pour revenir à l'héritage, et SHALL être persistée dans la section du workspace de `preferences.json` avec les autres surcharges du pool.

#### Scenario: Valeur héritée affichée
- **WHEN** Configuration ne définit pas de commande de validation et que le workspace n'a aucune surcharge
- **THEN** le champ indique que la commande est auto-détectée, sans valeur de surcharge

#### Scenario: Surcharge de la commande
- **WHEN** l'utilisateur saisit `cd backend && go test ./...` comme commande de validation du workspace et enregistre
- **THEN** la surcharge est persistée pour ce workspace, distinguée visuellement d'une valeur héritée, et appliquée aux workers de ce workspace uniquement

#### Scenario: Réinitialisation
- **WHEN** l'utilisateur réinitialise la surcharge de la commande de validation
- **THEN** le workspace hérite à nouveau de Configuration, ou de l'auto-détection si Configuration ne définit rien
