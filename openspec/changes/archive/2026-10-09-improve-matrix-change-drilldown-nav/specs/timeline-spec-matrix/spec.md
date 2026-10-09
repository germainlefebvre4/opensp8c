## MODIFIED Requirements

### Requirement: Drill-down vers le DetailPanel depuis le panel de spec
L'utilisateur SHALL pouvoir cliquer sur un change dans le panel de spec pour ouvrir le DetailPanel de ce change, remplaçant le panel de spec dans le slot droit. Le DetailPanel ouvert de cette manière SHALL afficher, au-dessus du titre du change, un bouton de retour libellé avec le nom de la spec d'origine (« ← nom-de-la-spec ») ramenant à la liste des changes de cette spec. Dans ce contexte, le bouton de fermeture (X) SHALL fermer le panel droit entier. Le DetailPanel affiché depuis le Kanban SHALL rester inchangé et ne pas afficher de bouton de retour.

#### Scenario: Clic sur un change dans le panel de spec
- **WHEN** l'utilisateur clique sur un change dans la liste du panel de spec
- **THEN** le panel droit bascule vers le DetailPanel de ce change (proposal, design, tasks) et son header affiche le bouton de retour « ← nom-de-la-spec »

#### Scenario: Retour au panel de spec
- **WHEN** l'utilisateur clique sur le bouton « ← nom-de-la-spec » du DetailPanel ouvert depuis le mode Matrice
- **THEN** le panel droit revient à la liste des changes de la spec précédemment sélectionnée, la spec reste sélectionnée dans la grille, et le DetailPanel n'est plus affiché

#### Scenario: Fermeture complète depuis le DetailPanel
- **WHEN** l'utilisateur clique sur le bouton de fermeture (X) du DetailPanel ouvert depuis le mode Matrice
- **THEN** le panel droit se ferme entièrement, aucune spec n'est sélectionnée et la grille reprend toute la largeur disponible

#### Scenario: Retour à la liste après une action sur le change
- **WHEN** l'utilisateur archive, supprime ou demande une correction pour le change affiché dans le DetailPanel ouvert depuis le mode Matrice, et que l'action aboutit
- **THEN** le panel droit revient à la liste des changes de la spec sélectionnée (et non à la fermeture complète du panel)

#### Scenario: Échap dans le DetailPanel
- **WHEN** le DetailPanel est ouvert depuis le mode Matrice, qu'aucune boîte de dialogue n'est ouverte et qu'aucun champ de saisie n'a le focus, et que l'utilisateur appuie sur Échap
- **THEN** le panel droit revient à la liste des changes de la spec sélectionnée

#### Scenario: Échap dans la liste des changes de la spec
- **WHEN** la liste des changes d'une spec est affichée dans le panel droit, qu'aucune boîte de dialogue n'est ouverte et qu'aucun champ de saisie n'a le focus, et que l'utilisateur appuie sur Échap
- **THEN** le panel droit se ferme et aucune spec n'est sélectionnée

#### Scenario: Échap ignoré pendant une saisie ou un dialogue
- **WHEN** une boîte de dialogue est ouverte ou un champ de saisie a le focus et que l'utilisateur appuie sur Échap
- **THEN** seul l'élément ayant le focus (dialogue ou champ) réagit ; la navigation du panel droit ne change pas

#### Scenario: Position de défilement conservée au retour
- **WHEN** l'utilisateur fait défiler la liste des changes d'une spec, ouvre le DetailPanel d'un change, puis revient à la liste
- **THEN** la liste est affichée à la même position de défilement qu'avant l'ouverture du DetailPanel

#### Scenario: Dernier change consulté mis en évidence
- **WHEN** l'utilisateur revient à la liste des changes d'une spec après avoir consulté un change
- **THEN** ce change est visuellement mis en évidence dans la liste, jusqu'à la sélection d'un autre change ou d'une autre spec, ou la fermeture du panel
