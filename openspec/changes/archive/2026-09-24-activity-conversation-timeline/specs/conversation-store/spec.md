# Spec Delta

## REMOVED Requirements

### Requirement: Affichage du log ff dans le DetailPanel
**Reason**: Remplacé par l'onglet **Conversation**, spécifié dans la capacité `activity-timeline`, qui fusionne tous les kinds de run (pas seulement `ff`) avec les entrées non-agent (cycle de vie Kanban, pool, git) dans un flux chronologique unique, au lieu d'un sélecteur de run manuel limité aux runs `ff`.
**Migration**: Les endpoints de listing et de récupération d'un run (`GET .../conversations/{kind}` et `GET .../conversations/{kind}/{ts}`) restent inchangés et continuent d'être utilisés en interne, désormais comme source de données lue par le nouvel endpoint fusionné `GET .../activity` d'`activity-timeline`. Aucune action utilisateur requise : l'onglet "Log" disparaît de l'interface au profit de l'onglet "Conversation" dans le même DetailPanel.

Le DetailPanel SHALL exposer un onglet **"Log"** listant les runs ff disponibles pour le changement ouvert. Le run le plus récent SHALL être sélectionné par défaut. Les messages SHALL être affichés en lecture seule dans le même format de rendu que les messages explore (réutilisation du renderer existant). Aucune zone de saisie n'est présente dans cet onglet.

#### Scenario: Onglet Log avec runs disponibles
- **WHEN** l'utilisateur ouvre le DetailPanel d'un changement ayant au moins un run ff
- **THEN** l'onglet "Log" est visible et affiche le run le plus récent par défaut

#### Scenario: Sélection d'un run antérieur
- **WHEN** l'utilisateur sélectionne un run plus ancien dans la liste
- **THEN** les messages de ce run s'affichent à la place du run courant

#### Scenario: Onglet Log sans runs
- **WHEN** le changement n'a encore aucun run ff
- **THEN** l'onglet "Log" est visible mais affiche un message vide ("Aucun run ff pour l'instant")
