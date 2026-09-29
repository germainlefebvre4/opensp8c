# Design

## Context

La locale de l'application n'existe que dans le navigateur (`localStorage.lang`, initialisée dans `frontend/src/i18n.ts`) ; le backend n'en a aucune notion. Les préférences globales vivent dans `backend/preferences.json`, lues/écrites par `internal/preferences/preferences.go` et exposées par `GET`/`PATCH /api/preferences` (`handlers/preferences.go`). Voir `proposal.md` - Why pour la motivation.

Tout lancement d'agent passe par `session.StartSubprocess` (`internal/session/subprocess.go`), appelé depuis : `session/manager.go` (exploration nommée, anonyme), `handlers/explore.go` (promotion d'une exploration), `handlers/ff.go`, `handlers/docs.go`, et `pool/worker.go` (via le seam `startSubprocessFn`). Chaque site charge lui-même les préférences une fois (`prefs.Load()`) pour construire `customEnv` via `p.EnvFor(agentID)` ; l'échec du chargement est déjà toléré (env vide). L'onglet « Langue » de `ConfigurationPage.tsx` (sélection par `?tab=language`) n'affiche aujourd'hui que `<LanguageSwitcher />`. Le mécanisme de transmission d'un prompt supplémentaire diffère selon l'agent :

| Agent | Sort actuel de `extraSystemPrompt` |
|---|---|
| Claude | `--append-system-prompt`, ré-appliqué à chaque lancement, reprise comprise |
| Codex, Copilot | reçoivent les mêmes arguments que Claude (placeholders, CLI non validées) |
| Gemini | ignoré ; process one-shot par tour, seul le préfixe `/opsx:explore ` est ajouté aux prompts non-slash |
| Antigravity | envoyé en premier message par `newAntigravityWriter`, puis supprimé à la reprise |

Contraintes de format : les artefacts OpenSpec ont un formalisme que `openspec validate` vérifie (titres `### Requirement:` / `#### Scenario:`, mots `SHALL`/`MUST`, `WHEN`/`THEN`). Les specs existantes de ce dépôt sont en français mais gardent ces marqueurs en anglais : la langue « documentation » ne doit donc s'appliquer qu'à la prose.

## Goals / Non-Goals

**Goals:**
- Un seul point où les langues sont résolues et où la consigne est construite, réutilisé par tous les sites de lancement.
- Transmission effective à chaque agent supporté, malgré des mécanismes d'injection hétérogènes.
- Modèle de données et liste de langues extensibles sans refonte.

**Non-Goals:**
- Refactorer la signature de `StartSubprocess` en une structure d'options, ou rendre le prompt de base spécifique au rôle (il reste celui de l'exploration pour tous les rôles ; hors périmètre, défaut préexistant).
- Traduire de façon rétroactive des conversations ou artefacts existants.
- Couvrir le tagger automatique (`claude -p`, il produit des tags kebab-case et non de la prose) ni l'archivage (aucun agent lancé).
- Étendre les langues de l'interface au-delà de EN/FR : la liste des langues d'agent et celle des langues d'UI sont deux ensembles distincts.

## Decisions

**Stockage : `agentLanguages` + `uiLocale` dans `preferences.json`.** Deux champs optionnels : `agentLanguages{chat, documentation, code}` (chaînes : `auto` ou un code) et `uiLocale`. Les champs absents équivalent aux défauts (`auto`/`auto`/`en`), donc aucune migration. Alternative écartée : stocker `auto` sous forme d'une copie figée de la locale — le réglage cesserait de suivre l'interface lors d'un changement EN/FR.

**Liste des langues : registre unique côté backend, exposée à l'UI.** Un petit paquet `internal/language` porte le registre (`code`, libellé natif affiché en UI, nom anglais utilisé dans la consigne), initialement `en` et `fr`. `GET /api/preferences` retourne aussi `supportedLanguages` ; le frontend construit ses trois listes à partir de cette réponse et ne duplique aucun code de langue. Le `PATCH` valide contre ce registre (400 sinon) et refuse `auto` pour `code`. Alternative écartée : liste codée dans le frontend — elle diverge du backend dès la troisième langue.

**Résolution : `language.Resolve(langs, uiLocale)` appelée à chaque lancement.** Prend des valeurs simples (et non `*Preferences`, pour que `preferences` puisse importer `language` pour la validation sans cycle) et retourne les trois langues concrètes : valeur explicite, sinon `uiLocale`, sinon `en`. Si `prefs.Load()` échoue, on résout avec les défauts et on lance l'agent quand même (le lancement ne dépend pas de la lisibilité des réglages). La résolution a lieu au lancement, pas à l'enregistrement : un agent en cours n'est jamais modifié.

**Synchronisation de `uiLocale` : hook frontend, pas requête par requête.** Un hook monté à la racine de l'application envoie `PATCH /api/preferences {uiLocale}` au chargement puis à chaque `languageChanged`. C'est le seul moyen de servir les workers du pool, lancés en arrière-plan sans requête du navigateur ; d'où le choix contre « envoyer la locale avec chaque requête de lancement ». Le hook est séparé de `i18n.ts` pour ne pas y importer le client HTTP.

**Consigne par rôle : `language.Directive(role, resolved)`.** Trois rôles : `Chat` (langue chat), `Docs` (langue documentation : génération de docs, fast-forward, promotion) et `Worker` (langue code pour le code, langue documentation pour les fichiers OpenSpec/docs modifiés). Le texte est court, en anglais, et formulé comme un défaut : « Unless the user explicitly asks otherwise, or the project's conventions (e.g. an existing i18n mechanism) require it, write ... in <Language> ». La consigne `Docs`/`Worker` précise que la langue s'applique à la prose et que le formalisme OpenSpec (titres requis, `SHALL`/`MUST`, `WHEN`/`THEN`) et les noms de fichiers restent inchangés. Le niveau `code` énumère explicitement identifiants, commentaires, messages de commit, messages d'erreur et textes visibles.

**Transmission : un paramètre `languageDirective` ajouté à `StartSubprocess`.** Le fusionner dans `extraSystemPrompt` suffirait pour Claude/Codex/Copilot, mais Gemini ignore ce prompt et Antigravity le perd à la reprise ; il faut donc que `StartSubprocess` connaisse la consigne séparément pour choisir le canal :
- Claude, Codex, Copilot : concaténée à `extraSystemPrompt` (donc ré-appliquée à chaque lancement, reprise incluse).
- Antigravity : ajoutée au cadrage du premier message, et envoyée seule à la reprise (aujourd'hui le cadrage est supprimé à la reprise).
- Gemini : ajoutée à chaque tour, dans le message.

Alternative écartée : passer par des variables d'environnement — aucun agent ne les lit comme une instruction. Alternative reportée : transformer les paramètres positionnels (9 aujourd'hui, 10 avec celui-ci) en structure d'options ; plus propre, mais hors périmètre et conflictuel avec `per-agent-cli-env-config`, qui modifie les mêmes sites.

**Sites de lancement : une méthode sur les préférences déjà chargées.** Chaque site tient déjà un `*Preferences` chargé pour `EnvFor` ; on y ajoute `(*Preferences).LanguageDirective(role)`, sûre sur un pointeur `nil` (échec de chargement : consigne par défaut). Pas de seconde lecture du fichier, et le motif est celui d'`EnvFor`. Aucun site ne construit de consigne lui-même. Alternative écartée : un helper prenant le service de préférences, qui relirait le fichier à chaque lancement.

**Interface : trois listes dans l'onglet Langue.** Un composant d'onglet remplace le `<LanguageSwitcher />` nu actuel et le conserve en tête. Alimentées par `supportedLanguages`, enregistrées avec le `PATCH` existant via `usePatchPreferences`. Les listes chat et documentation ajoutent l'option `auto` qui affiche la langue résolue ; la liste code n'a pas `auto`.

## Risks / Trade-offs

- [Gemini/Antigravity : une consigne ajoutée à un prompt commençant par une slash-command (`/opsx:ff <change>`, `/opsx:apply <change>`) peut être lue comme argument de la commande] → placer la consigne dans un paragraphe séparé après la commande et le vérifier sur chaque CLI (tâche dédiée) ; si l'argument est corrompu, repli à valider (ex. consigne envoyée dans un tour préalable) sans changer les specs.
- [Codex et Copilot reçoivent des arguments de type Claude qui ne sont pas validés] → la consigne suit ce canal aujourd'hui ; si leurs vraies interfaces sont branchées plus tard, ces agents devront être reclassés dans le tableau de transmission.
- [`uiLocale` est global : le dernier navigateur ouvert gagne] → acceptable pour un outil local mono-utilisateur ; noté comme limite.
- [Un modèle peut ignorer ou dérouler partiellement la consigne, surtout sur une conversation reprise déjà écrite dans une autre langue] → consigne courte et explicite, ré-appliquée à chaque lancement ; le réglage reste un défaut, pas une garantie.
- [Les specs delta de `platform-configuration` côtoient l'exigence « Sous-onglet Langue dans Configuration »] → elles ne portent que sur des requirements ajoutés (`ADDED`) et n'en modifient aucun existant ; aucun recalage n'est nécessaire à l'archivage.

## Migration Plan

Aucune migration de données : les deux nouveaux champs sont optionnels et absents des `preferences.json` existants. Déploiement en une fois (backend + frontend). Retour arrière : retirer les champs ; un `preferences.json` qui les contient reste lisible par une version antérieure, qui les ignore.

## Open Questions

- Formulation exacte de la consigne par rôle : à ajuster à l'usage, sans effet sur les specs ni sur le découpage des tâches.
- Libellés natifs dans le registre (« Français ») : affichés tels quels ; une localisation de ces libellés n'est pas nécessaire tant que la liste reste courte.
