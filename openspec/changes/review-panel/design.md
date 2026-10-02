# Design

## Context

Rien dans l'application ne permet de lire ce qu'une branche `feature/<change>` apporte : `WorktreeController` ne sait ni lister les fichiers modifiés ni produire un patch, et le routeur n'expose aucun endpoint de revue. Côté frontend, le `DetailPanel` fonctionne par onglets (`Tab`: tasks, proposal, design, conversation, tags, actions) avec un toggle vue source/rendu sur proposal et design ; `SpecEditor` rend déjà un diff avec la lib `diff` (`diffLines` sur deux textes, lignes vertes/rouges avec `+`/`-`) ; `AgentsPage` sélectionne un run via un paramètre d'URL et affiche `AgentRunPanel`. Voir `proposal.md` - Why.

Contraintes :
- La revue doit rester lisible quand le pool est arrêté, voire quand le worktree a disparu : seules la branche et le dépôt comptent.
- Les chemins viennent du client : ils ne doivent jamais atteindre git sans validation.
- Le statut `to-review` (change `review-state`) conditionne l'accès ; ce change n'écrit rien.

## Goals / Non-Goals

**Goals:**
- Lire la revue d'un change (fichiers, patch, contenu final) via trois endpoints en lecture seule, indépendants du worktree.
- Un onglet « Revue » cohérent avec les onglets existants (composants, i18n, SSE).
- Un diff qui correspond à ce que la fusion apporterait réellement.

**Non-Goals:**
- Approuver ou demander une correction (change `review-actions`).
- Commentaires par ligne, édition du code, diff côte à côte, recherche dans le diff.
- Détection de renommages (un renommage apparaît comme suppression + ajout).
- Virtualisation des gros diffs : les limites de taille et le dépliage à la demande suffisent.

## Decisions

**1. Diff contre le point de divergence, calculé sur les références git.** Les trois endpoints s'appuient sur `git diff --no-renames <base>...feature/<change>` (trois points : le point de divergence, non la pointe de la base), de sorte que les commits ajoutés à `main` depuis ne polluent pas la revue. La base est `BaseBranch(change)` (branche enregistrée à la création de la branche), à défaut `CurrentBranch()` ; si la base enregistrée n'existe plus comme référence, repli sur `HEAD`, et le nom de base effectivement utilisé est renvoyé. Aucune lecture du worktree : la revue marche pool arrêté et worktree supprimé. `target_ahead` réutilise `TargetAhead(change)`. Alternative écartée : *git diff dans le worktree* (échoue si le worktree est absent, et expose un état non committé).

**2. Liste des fichiers : `--numstat -z` + `--name-status -z`.** Deux appels, fusionnés par chemin, parsés avec le séparateur NUL pour supporter les noms avec espaces ou caractères spéciaux. `numstat` marque les binaires par `-`/`-` → `binary: true`. `--no-renames` garde les statuts à `A`/`M`/`D`, mappés sur `added`/`modified`/`deleted`.

**3. Validation du chemin par appartenance.** Le paramètre `path` n'est accepté que s'il figure dans la liste des fichiers de la revue (calculée à la requête). Les chemins vides, absolus ou contenant `..` sont rejetés avant tout appel git (`400`) ; un chemin valide mais hors revue donne `404`. Git reçoit toujours `--` avant le chemin, ou la forme `feature/<change>:<chemin>` pour `show`. Aucun chemin client n'est interprété comme option ou comme référence arbitraire.

**4. Limites de taille.** Patch et contenu sont lus avec une borne de 512 Kio ; au-delà, la réponse est tronquée avec `truncated: true`. Pour le contenu, la taille est connue avant lecture (`cat-file -s`), pour éviter de charger un gros blob. Un fichier binaire ne renvoie ni patch ni contenu.

**5. Contenu final pour le rendu Markdown.** Un troisième endpoint (`review/file`) renvoie le fichier à sa version dans la branche (`git show`), pour que le toggle diff/rendu réutilise le composant `Markdown` existant, comme le toggle de proposal et design. Alternative écartée : *rendre le patch en Markdown*, qui n'aurait pas de sens.

**6. Frontend : un composant `ReviewTab`, des hooks nichés sous la clé du détail.** Le patch unifié est parsé avec `parsePatch` de la lib `diff` déjà installée et rendu avec le même style de lignes que `SpecEditor` (vert/rouge, `+`/`-`), complété par les numéros de lignes des hunks ; si l'extraction d'un petit composant de lignes partagé est simple, `SpecEditor` l'adopte, sinon le style est dupliqué sans toucher à `SpecEditor`. Les hooks (`useChangeReview`, `useReviewDiff`, `useReviewFile`) utilisent des clés de requête imbriquées sous celle du détail du change, de sorte que l'invalidation existante à `change_updated` rafraîchit aussi la revue ; les diffs et contenus ne sont chargés qu'au dépliage. Les fichiers sont groupés par préfixe `openspec/` (groupe OpenSpec) ou non (groupe Code).

**7. Onglet conditionnel et repli.** `Tab` gagne la valeur `review`, ajoutée à la barre d'onglets uniquement si `kanban_status === 'to-review'`. Si le change quitte ce statut alors que l'onglet est actif (approbation, correction), un effet remet `activeTab` à `tasks`. L'onglet ne devient pas l'onglet par défaut : le comportement d'ouverture du panneau est inchangé.

**8. Lien vers le run.** Le dernier run du change est le plus récent de `usePoolRuns` filtré sur le change. Le lien navigue vers la vue Agents avec le paramètre de sélection de run existant (format et nom du paramètre exportés depuis `AgentsPage` ou un module partagé pour éviter une copie).

## Risks / Trade-offs

- [Un chemin forgé ou un nom de fichier piégé atteint git] → validation en deux temps (forme, puis appartenance à la liste), `--` systématique, sortie NUL-séparée, tests dédiés (`..`, absolu, nom commençant par `-`, espaces).
- [Gros diff ou gros binaire fait ramer le serveur ou le navigateur] → limites de 512 Kio, dépliage à la demande, taille connue avant lecture ; pas de virtualisation (non-objectif).
- [Deux appels git par liste de fichiers] → coût faible, calcul à la demande seulement à l'ouverture de l'onglet ; pas de cache.
- [Base enregistrée disparue ou HEAD détaché] → repli sur `HEAD` avec le nom de base utilisé renvoyé dans la réponse ; un cas sans repli possible renvoie une erreur lisible.
- [Renommages vus comme suppression + ajout] → choix assumé pour la simplicité et la robustesse du parsing ; réévaluable.
- [Spec contradictoire si `review-actions` est archivé avant ce change] → ce change retire l'ancien requirement de `agent-pool-ui` ; archiver ce change avant ou avec `review-actions` (rappelé dans les deux proposals).
