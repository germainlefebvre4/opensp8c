# Spec Delta

## ADDED Requirements

### Requirement: Modale de lancement pré-remplie avec la configuration résolue
La modale de configuration et de lancement du pool SHALL être pré-remplie avec la configuration résolue du workspace (défauts de Configuration et surcharges du workspace) pour la taille du pool, le mode de délégation et le nombre de tentatives. L'utilisateur SHALL pouvoir ajuster ces valeurs pour le lancement en cours sans modifier les valeurs enregistrées.

#### Scenario: Pré-remplissage avec une surcharge
- **WHEN** le workspace surcharge la taille du pool à 4 et que l'utilisateur ouvre la modale de lancement
- **THEN** le sélecteur de taille affiche 4 et le mode affiché est celui résolu pour ce workspace

#### Scenario: Ajustement ponctuel
- **WHEN** l'utilisateur change la taille à 2 dans la modale et clique sur « Démarrer »
- **THEN** le pool démarre avec une taille de 2
- **THEN** la surcharge enregistrée du workspace reste à 4

#### Scenario: Workspace sans surcharge
- **WHEN** le workspace n'a aucune surcharge et que Configuration fixe la taille par défaut à 2
- **THEN** la modale affiche une taille de 2
