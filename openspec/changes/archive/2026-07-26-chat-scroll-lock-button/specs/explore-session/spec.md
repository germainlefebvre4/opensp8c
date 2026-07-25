## ADDED Requirements

### Requirement: Scroll-Lock et bouton "Défiler vers le bas" dans le panneau d'exploration
Le panneau d'exploration nommé SHALL gérer intelligemment le défilement de la liste des messages lors de la réception de réponses en streaming. Si l'utilisateur fait défiler la vue vers le haut (s'il n'est plus en bas de la liste), le défilement automatique SHALL être désactivé (verrouillé) afin de permettre la lecture sans interruption. Un bouton flottant "Défiler vers le bas" (Scroll to bottom) SHALL alors apparaître. S'il clique sur ce bouton ou s'il envoie un nouveau message, la liste SHALL défiler doucement jusqu'en bas et le défilement automatique SHALL être réactivé.

#### Scenario: Défilement automatique actif au bas de la liste
- **WHEN** l'utilisateur est au bas de la liste des messages (ou s'il n'y a pas de scrollbar) et que l'assistant écrit ou qu'un nouveau message arrive
- **THEN** la liste des messages défile automatiquement vers le bas de manière fluide pour afficher les derniers mots en temps réel

#### Scenario: Activation du Scroll-Lock
- **WHEN** l'utilisateur remonte manuellement dans la liste des messages (s'éloignant du bas de plus de 20px) pendant que l'assistant écrit
- **THEN** le défilement automatique est immédiatement verrouillé, le viewport reste fixe sur la zone de lecture de l'utilisateur, et un bouton flottant avec une icône de flèche vers le bas s'affiche en bas à droite de la zone des messages

#### Scenario: Clic sur le bouton de défilement vers le bas
- **WHEN** le bouton de défilement vers le bas est visible et que l'utilisateur clique dessus
- **THEN** la liste défile de manière fluide jusqu'en bas, le bouton flottant disparaît, et le défilement automatique de la session est réactivé

#### Scenario: Envoi d'un message force le défilement vers le bas
- **WHEN** l'utilisateur envoie un nouveau message dans le chat alors qu'il avait fait défiler la vue vers le haut
- **THEN** la liste défile immédiatement jusqu'en bas pour afficher son message, le bouton flottant disparaît, et le défilement automatique est réactivé
