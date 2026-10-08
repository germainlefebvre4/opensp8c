# Spec Delta

## ADDED Requirements

### Requirement: Pilote affiché dans le rapport de vérification UI
Lorsque le bandeau de vérification du DetailPanel affiche le rapport d'une étape `ui`, il SHALL indiquer le pilote utilisé (`driver` du rapport) par un libellé court : « automatique » pour `auto`, « Playwright » pour `playwright`, « navigateur Chrome » pour `chrome` et « personnalisé » pour `custom`. Pour `chrome`, le libellé SHALL être accompagné d'une mention signalant que le navigateur de l'utilisateur a été piloté. Lorsque le rapport ne porte pas de pilote, aucun libellé de pilote ne SHALL être affiché.

#### Scenario: Pilote Playwright
- **WHEN** l'utilisateur ouvre un change dont le rapport UI porte `driver` à `playwright`
- **THEN** le bandeau affiche « Playwright »

#### Scenario: Pilote Chrome
- **WHEN** le rapport UI porte `driver` à `chrome`
- **THEN** le bandeau affiche « navigateur Chrome » avec la mention que le navigateur de l'utilisateur a été piloté

#### Scenario: Rapport sans pilote
- **WHEN** le rapport affiché est celui de la conformité ou celui d'un run antérieur sans `driver`
- **THEN** aucun libellé de pilote n'est affiché
