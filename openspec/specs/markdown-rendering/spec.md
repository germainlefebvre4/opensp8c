# markdown-rendering Specification

## Purpose

Définit le contrat de rendu Markdown commun à tous les panneaux de l'interface (documentation, specs, détail de change, exploration, runs d'agent), afin que tableaux, code inline et blocs de code s'affichent de manière lisible et identique partout.

## Requirements

### Requirement: Rendu des tableaux Markdown
Tout contenu Markdown affiché dans l'interface SHALL interpréter la syntaxe de tableau GFM (en-tête, ligne de séparation `|---|`, lignes de données) et la rendre comme un tableau visuel avec bordures et en-têtes distincts, plutôt que comme du texte brut. Un tableau plus large que son conteneur SHALL défiler horizontalement sans élargir la page.

#### Scenario: Tableau GFM valide
- **WHEN** un contenu Markdown contient un tableau avec une ligne d'en-tête, une ligne de séparation et des lignes de données
- **THEN** un tableau avec en-têtes et cellules délimitées est affiché, sans pipes visibles

#### Scenario: Cellule contenant du Markdown inline
- **WHEN** une cellule d'un tableau contient du gras, de l'italique ou du code inline
- **THEN** la cellule affiche ce contenu mis en forme

#### Scenario: Tableau plus large que le conteneur
- **WHEN** un tableau dépasse la largeur disponible du panneau
- **THEN** le tableau défile horizontalement dans son conteneur et la mise en page du panneau n'est pas déformée

### Requirement: Rendu du code inline
Le code inline (texte entouré d'un seul backtick) SHALL être affiché avec une police monospace, un fond gris et un arrondi, sans afficher les backticks délimiteurs.

#### Scenario: Code inline dans un paragraphe
- **WHEN** un paragraphe contient `openspec/`
- **THEN** `openspec/` est affiché en police monospace sur fond gris, sans backtick visible

#### Scenario: Code inline dans un tableau ou une liste
- **WHEN** du code inline apparaît dans une cellule de tableau ou un élément de liste
- **THEN** il reçoit le même style que dans un paragraphe

### Requirement: Distinction entre code inline et bloc de code
Un bloc de code délimité par des triples backticks, avec ou sans langage, SHALL être rendu comme un bloc distinct du code inline : le style du code inline (fond gris d'enrobage, arrondi) ne SHALL PAS s'appliquer à son contenu.

#### Scenario: Bloc de code sans langage
- **WHEN** un contenu Markdown contient un bloc délimité par ``` sans langage
- **THEN** il est affiché comme un bloc de code multiligne, sans le style du code inline appliqué à chaque ligne

#### Scenario: Bloc de code avec langage
- **WHEN** un contenu Markdown contient un bloc ```ts
- **THEN** il est affiché comme un bloc de code multiligne, distinct visuellement du code inline

### Requirement: Cohérence du rendu entre panneaux
Les panneaux affichant du Markdown rendu (documentation, specs brutes, détail de change, exploration nommée et anonyme, runs d'agent) SHALL appliquer les mêmes règles de rendu des tableaux, du code inline et des blocs de code.

#### Scenario: Même contenu dans deux panneaux
- **WHEN** un même contenu Markdown contenant un tableau et du code inline est affiché dans deux panneaux différents
- **THEN** le tableau et le code inline y sont rendus de la même manière
