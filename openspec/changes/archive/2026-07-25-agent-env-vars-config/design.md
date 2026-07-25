## Context

Les agents de code CLI (en particulier Gemini CLI dans les environnements d'entreprise) requièrent plusieurs variables d'environnement clés pour fonctionner, comme `GOOGLE_CLOUD_PROJECT` (pour l'authentification GCP), `GEMINI_MODEL` (pour spécifier le modèle d'IA) ou `GEMINI_SANDBOX` (pour contrôler l'exécution sécurisée).
Actuellement, pour injecter de nouvelles variables d'environnement, l'utilisateur doit arrêter le serveur backend `opensp8c` et le relancer dans un shell contenant ces variables. Cela nuit à l'expérience développeur "à chaud" et limite l'utilisation de multiples projets GCP ou configurations de modèles.

## Goals / Non-Goals

**Goals:**
- Permettre à l'utilisateur de configurer ses variables d'environnement de manière interactive à chaud via l'interface Web.
- Sauvegarder les paires Clé/Valeur de l'environnement personnalisé dans `preferences.json` sous un dictionnaire `env`.
- Injecter dynamiquement ces variables dans les subprocesses de tous les agents lancés (`StartSubprocess`) en fusionnant les variables personnalisées avec les variables système existantes (`os.Environ()`).
- Fournir une UI ergonomique (bouton de configuration, boîte de dialogue/modal) pré-configurée avec des indications claires pour les variables courantes de Gemini CLI.

**Non-Goals:**
- Masquer ou chiffrer les variables d'environnement dans le fichier local `preferences.json` (ce fichier étant déjà local à l'utilisateur).
- Synchroniser l'environnement entre plusieurs machines ou stocker les variables d'environnement côté cloud.

## Decisions

### Décision 1 : Fusionner au lieu d'écraser l'environnement parent
Lors de l'attribution de `cmd.Env`, si nous utilisons uniquement nos variables personnalisées, le subprocess ne pourra plus localiser les exécutables de base (faute de `PATH`) ni les dossiers de configuration utilisateur (faute de `HOME` ou `USER`).
- **Option A (Remplacement complet) :** Assigner uniquement `p.Env` à `cmd.Env`. (Rejeté : casse le fonctionnement des exécutables CLI).
- **Option B (Fusion et surcharge - Sélectionnée) :** Charger l'environnement parent via `os.Environ()`, y injecter ou écraser les clés fournies par l'utilisateur, puis assigner le résultat à `cmd.Env`.

### Décision 2 : Point d'accès UI contextualisé
- **Option A (Page globale de configuration) :** Ajouter un onglet "Paramètres globaux" dans l'application. (Rejeté : trop déconnecté du sélecteur d'agent).
- **Option B (Bouton d'engrenage dans la barre latérale - Sélectionnée) :** Ajouter un bouton d'engrenage à côté du sélecteur d'agent au bas de la barre latérale. Cela maintient la configuration des variables d'environnement proche du choix de l'agent actif.

### Décision 3 : Structure des entrées de variables recommandées
Pour faciliter la vie de l'utilisateur, nous fournirons des champs spécifiques et documentés pour les variables Gemini courantes, tout en permettant l'ajout illimité de paires Clé/Valeur arbitraires pour les autres agents ou configurations avancées.

## Risks / Trade-offs

- **[Risque] Saisie de clés d'API ou secrets sensibles**
  - *Mitigation :* Bien que `preferences.json` soit local, nous devons nous assurer que les inputs de saisie ne révèlent pas nécessairement les clés en clair si l'utilisateur souhaite les masquer, ou inclure des indications claires. Nous conseillons de ne jamais commiter `preferences.json` contenant des secrets d'entreprise réels (le fichier devant rester local).
