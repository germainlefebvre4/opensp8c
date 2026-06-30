## Context

La SpecsPage affiche actuellement le contenu d'une spec sélectionnée (markdown rendu + TOC). Le backend retourne uniquement `name` et `content` pour chaque spec — sans aucun lien avec les changes qui les ont produites ou modifiées.

Or chaque change archivé ou actif contient un dossier `specs/` listant exactement les specs qu'il a créées ou modifiées. Cette relation existe déjà dans le filesystem ; elle n'est juste pas exposée ni affichée.

## Goals / Non-Goals

**Goals:**
- Exposer un endpoint `/specs/overview` retournant l'index inversé `spec → changes[]` (actifs + archivés)
- Ajouter un toggle Contenu/Historique sur la SpecsPage existante
- En mode Historique, afficher la timeline complète des changes par spec inline
- Permettre d'ouvrir le DetailPanel d'un change depuis cette timeline (réutilisation du composant existant)
- Mettre en évidence les specs sans aucun change lié et les specs orphelines

**Non-Goals:**
- Écrire dans les fichiers du projet applicatif (lecture seule)
- Gestion de statut de spec (approved/draft/stale)
- Affichage inline du delta spec d'un change (le clic ouvre le DetailPanel complet)
- Options de tri ou filtre sur la timeline
- Pagination ou virtualisation (28 specs, charge négligeable)

## Decisions

**1. Nouveau endpoint `/specs/overview` plutôt qu'enrichir `/specs`**
Le shape de données est fondamentalement différent : index inversé, pas de contenu, champs de traçabilité. Enrichir `/specs` forcerait tous les clients à porter des données inutiles. Un endpoint dédié garde le contrat propre.

**2. Toggle local state côté frontend (pas de route dédiée)**
Le mode Contenu/Historique est un état UI transitoire. Aucun lien profond nécessaire, pas de modification du routing React Router existant.

**3. Extraction de date depuis le nom du change**
Les changes archivés suivent la convention `YYYY-MM-DD-<slug>`. La date est extraite par regex au backend. Pour les changes actifs (sans préfixe date), fallback sur le champ `created` du `.openspec.yaml`. Le slug affiché = nom sans préfixe date dans les deux cas.

**4. Calcul de l'index inversé à la demande (pas de cache)**
28 specs, 31 changes : la construction de l'index inversé en mémoire à chaque appel est négligeable. Pas de cache ni de watch filesystem nécessaires à ce stade.

**5. DetailPanel réutilisé sans modification**
`GetChangeDetail` cherche d'abord dans `changes/`, puis dans `changes/archive/`. Les changes archivés sont déjà supportés. En mode Historique, le slot droit de la SpecsPage (actuellement occupé par le contenu + TOC) accueille le DetailPanel.

## Risks / Trade-offs

- **Convention de nommage évolutive** : si un change actif ne suit pas le format `YYYY-MM-DD-<slug>`, la date provient de `.openspec.yaml`. Si ce fichier est absent ou mal formé, la date sera vide. → Mitigation : afficher "—" pour les dates manquantes, pas d'erreur.
- **Specs orphelines** : une spec peut être référencée dans `changes/*/specs/` sans avoir de dossier correspondant dans `openspec/specs/` (sync incomplète). → Surfacé dans le champ `orphans[]` de la réponse, affiché en bas de la vue Historique.
- **Couplage au layout du DetailPanel** : le DetailPanel est conçu pour le Kanban. Certaines actions (archive, FF) n'ont pas de sens depuis la vue Specs. → Le composant affiche déjà ces actions de façon conditionnelle ; pas d'adaptation nécessaire à court terme.
