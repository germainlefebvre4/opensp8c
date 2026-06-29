## 1. Backend — Package preferences

- [ ] 1.1 Créer `internal/preferences/preferences.go` avec struct `Preferences` (`DefaultAgent string`, `SessionAgents map[string]string`)
- [ ] 1.2 Implémenter `Load(path string) (*Preferences, error)` avec création du fichier par défaut si absent
- [ ] 1.3 Implémenter `Save(path string, prefs *Preferences) error` avec mutex pour les accès concurrents
- [ ] 1.4 Exposer le chemin du fichier via variable d'environnement `PREFERENCES_PATH` avec fallback à côté de `CONFIG_PATH`

## 2. Backend — Package agents

- [ ] 2.1 Créer `internal/agents/agents.go` avec struct `AgentConfig` (`ID`, `Label`, `CLI`, `VersionArgs []string`) et liste statique des 5 agents
- [ ] 2.2 Implémenter `Detect(agent AgentConfig) AgentStatus` : probe `which <cli>` et `<cli> <versionArgs>` pour retourner `installed` + `version`
- [ ] 2.3 Implémenter `DetectAll() []AgentStatus` pour prober tous les agents en parallèle
- [ ] 2.4 Gérer le cas spécifique de Copilot (`gh copilot --version`)

## 3. Backend — Endpoints API

- [ ] 3.1 Ajouter `GET /api/agents` dans le router — appelle `DetectAll()` et retourne la liste JSON
- [ ] 3.2 Ajouter `GET /api/preferences` — retourne `{ defaultAgent }`
- [ ] 3.3 Ajouter `PATCH /api/preferences` — valide l'ID agent, met à jour preferences.json
- [ ] 3.4 Injecter `PreferencesService` dans `ExploreHandler` via le constructeur

## 4. Backend — CLI router dans subprocess

- [ ] 4.1 Modifier `subprocess.go` : remplacer le `exec("claude", ...)` hardcodé par un `AgentConfig` passé en paramètre
- [ ] 4.2 Mettre à jour `Subprocess struct` et `NewSubprocess(ctx, agentConfig, systemPrompt)` pour accepter la config agent
- [ ] 4.3 Ajouter les args CLI propres à chaque agent (à valider pour Codex, Gemini, Antigravity, Copilot)

## 5. Backend — Résolution de l'agent dans le session manager

- [ ] 5.1 Modifier `manager.go` : `Start(workspaceID, changeName)` résout l'agent depuis `sessionAgents` ou `defaultAgent`
- [ ] 5.2 Modifier `StartAnonymous()` : résout l'agent depuis `defaultAgent` uniquement
- [ ] 5.3 Implémenter le fallback Claude si l'agent mémorisé n'est plus installé + injection d'un message d'avertissement dans la session
- [ ] 5.4 Écrire `sessionAgents[workspaceID/changeName]` dans preferences.json à la création d'une named session

## 6. Frontend — Sélecteur d'agent global

- [ ] 6.1 Créer le composant `AgentSelector.tsx` : dropdown au-dessus des workspaces dans le menu gauche
- [ ] 6.2 Appeler `GET /api/agents` au montage pour obtenir la liste avec statut d'installation
- [ ] 6.3 Griser et désactiver les agents non installés dans le dropdown
- [ ] 6.4 Afficher la version à côté du nom de chaque agent installé
- [ ] 6.5 Au changement de sélection : appeler `PATCH /api/preferences` et mettre à jour l'état local

## 7. Frontend — Badge agent dans les conversations

- [ ] 7.1 Modifier `ExplorePanel.tsx` : remplacer le `assistantName = 'Claude'` hardcodé par une prop `agentLabel` passée depuis le parent
- [ ] 7.2 Modifier `ExploreAnonymousPanel.tsx` : même ajustement
- [ ] 7.3 Modifier `TypingBubble.tsx` : utiliser `agentLabel` en prop plutôt que la constante
- [ ] 7.4 Ajouter le badge agent + version dans l'en-tête de `ExplorePanel` et `ExploreAnonymousPanel`
- [ ] 7.5 Faire passer l'info agent depuis les hooks `useExploreSession` / données de session vers les panels

## 8. Frontend — Intégration API préférences

- [ ] 8.1 Ajouter les fonctions `getPreferences()` et `patchPreferences()` dans `src/lib/api.ts`
- [ ] 8.2 Créer un hook `useAgentPreferences()` avec React Query pour lire/écrire les préférences
- [ ] 8.3 Initialiser le sélecteur global avec la valeur lue depuis `GET /api/preferences` au chargement
