# Spec Delta

## ADDED Requirements

### Requirement: Journalisation de l'exécution de chaque worker
Chaque worker du pool SHALL consigner son exécution dans un run persistant de kind `pool` (voir `agent-run-detail`) tout au long de la vie de son subprocess d'agent, y compris les tours de correction envoyés lors de la boucle d'auto-guérison. Cette journalisation ne SHALL PAS modifier le déroulement de l'exécution : un échec d'écriture du journal SHALL NOT interrompre ni mettre en pause le worker.

#### Scenario: Tours de guérison consignés
- **WHEN** la validation échoue et que le worker renvoie les erreurs à l'agent par un tour de suivi
- **THEN** ce tour et la réponse de l'agent sont ajoutés au même run que la tentative initiale

#### Scenario: Échec d'écriture du journal
- **WHEN** l'écriture d'une ligne dans le run du worker échoue
- **THEN** le worker poursuit son exécution normalement

### Requirement: État des workers cohérent sous concurrence
Le statut et l'aperçu d'activité d'un worker SHALL être lus et écrits de façon synchronisée avec les instantanés retournés par le statut du pool, de sorte qu'un instantané ne présente jamais un état partiellement mis à jour ni ne provoque de course de données.

#### Scenario: Instantané pendant une mise à jour de l'activité
- **WHEN** le statut du pool est demandé pendant que le worker met à jour son aperçu d'activité
- **THEN** la réponse contient un état cohérent du worker et le détecteur de courses de données ne signale rien
