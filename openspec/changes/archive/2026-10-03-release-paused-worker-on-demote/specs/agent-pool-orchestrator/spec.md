## MODIFIED Requirements

### Requirement: Changes en pause exclues du dispatcher
Tant qu'un worker est en pause pour un changement, le dispatcher SHALL NOT réassigner ce changement à un worker, même s'il reste dans la colonne Todo et sans dépendance en attente. Le changement SHALL redevenir éligible uniquement lorsque sa pause est levée par une reprise explicite, par la rétrogradation du changement vers Ready (voir `kanban-ready-column`) ou par l'arrêt du pool. Les autres changements éligibles SHALL continuer à être distribués normalement. Un worker dont l'exécution est interrompue par l'arrêt du pool ou par l'annulation explicite de son changement SHALL NOT apparaître comme en pause, et un pool redémarré SHALL partir sans aucune pause héritée d'un run précédent. L'identifiant d'un worker terminé SHALL NOT pouvoir supprimer l'état d'un worker plus récent qui a réutilisé cet identifiant.

#### Scenario: Change en pause non relancée
- **WHEN** le worker du changement `add-user-auth` passe en pause alors que ce changement reste éligible dans la colonne Todo
- **THEN** le dispatcher ne lui assigne aucun nouveau worker aux ticks suivants et aucune nouvelle tentative de provisionnement n'a lieu

#### Scenario: Autres changements non bloqués
- **WHEN** le changement A est en pause et que le changement B est éligible avec un worker libre
- **THEN** le dispatcher distribue B normalement

#### Scenario: Arrêt du pool
- **WHEN** l'utilisateur arrête le pool alors que le changement A est en pause, puis le redémarre
- **THEN** le changement A redevient éligible à la distribution

#### Scenario: Arrêt du pool pendant l'exécution d'un worker
- **WHEN** l'utilisateur arrête le pool alors que le worker du changement A exécute un tour d'agent, puis redémarre le pool
- **THEN** le worker interrompu n'apparaît pas comme en pause, le changement A est éligible dès le redémarrage et son issue enregistrée est `stopped`

#### Scenario: Annulation explicite d'un changement
- **WHEN** l'utilisateur désarme de force le changement A alors que son worker est actif
- **THEN** le worker est annulé sans apparaître comme en pause

#### Scenario: Pause levée par la rétrogradation du changement
- **WHEN** l'utilisateur rétrograde le changement A vers Ready alors que son worker est en pause, puis le promeut de nouveau vers Todo
- **THEN** le worker en pause a disparu de la liste des workers du pool dès la rétrogradation, et le dispatcher redistribue A aux ticks suivants sans reprise explicite

#### Scenario: Libération d'une pause sans effet sur les autres workers
- **WHEN** la pause du changement A est levée par sa rétrogradation alors que le worker du changement B s'exécute
- **THEN** le worker de B n'est ni interrompu ni modifié, et son identifiant reste inchangé
