# Spec Delta

## MODIFIED Requirements

### Requirement: Panneau d'état du pool actif
Lorsqu'un pool est en cours d'exécution, l'utilisateur SHALL pouvoir consulter un panneau listant chaque worker actif avec le change auquel il est assigné, son statut (`idle`, `working`, `testing`, `healing`, ou `paused`), et un aperçu de l'activité en cours de l'agent (dernière sortie produite par sa session réelle). Ce panneau SHALL exposer une action "Stop Pool" qui arrête l'ensemble du pool. Ce panneau ne fournit pas d'action d'arrêt individuelle par worker.

#### Scenario: Affichage de la liste des workers actifs
- **WHEN** l'utilisateur ouvre le panneau d'état alors que 2 workers sur 3 sont assignés à des changes
- **THEN** le panneau affiche les 2 workers actifs avec le nom de leur change assigné et leur statut courant, et indique que le 3e worker est disponible (idle ou non affiché comme assigné)

#### Scenario: Arrêt du pool depuis le panneau
- **WHEN** l'utilisateur clique sur "Stop Pool" dans le panneau d'état
- **THEN** l'application envoie la requête d'arrêt au backend, le panneau se ferme, et le bouton d'en-tête repasse à l'état "Lancer le Pool"

#### Scenario: Aperçu de l'activité en cours d'un worker
- **WHEN** un worker est au statut `working` sur le changement `add-user-auth` et que sa session agent produit de la sortie
- **THEN** le panneau affiche, pour ce worker, un aperçu texte de la dernière activité de l'agent, mis à jour au fil de l'exécution
