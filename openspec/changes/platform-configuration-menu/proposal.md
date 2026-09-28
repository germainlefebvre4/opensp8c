# Proposal

## Why

Les réglages liés aux agents et aux binaires CLI sont aujourd'hui éparpillés et peu découvrables : les variables d'environnement recommandées (Gemini) et le mode question native ne vivent que dans `AgentSettingsModal`, ouverte via une petite icône engrenage à côté du sélecteur d'agent dans la sidebar workspace ; la liste des agents supportés et leur statut d'installation n'apparaît que dans le dropdown du même sélecteur. Rien ne permet de visualiser le registre des agents de la plateforme de façon dédiée, et l'entrée d'accès à la configuration CLI est visuellement rattachée à un workspace alors que ces réglages sont globaux à la machine.

## What Changes

- Ajout d'une nouvelle entrée de navigation globale **Configuration**, positionnée à part de la liste d'onglets existante (Kanban/Specs/Timeline/Agents/Réglages) et sans paramètre `?workspace=`, pour bien marquer qu'elle n'est pas rattachée à un workspace.
- La page Configuration expose deux sections :
  - **Agents** : vue en lecture seule du registre des agents supportés par la plateforme (claude, codex, gemini, antigravity, copilot) avec statut d'installation et version détectée (réutilise `GET /api/agents`, déjà utilisé par le dropdown sidebar).
  - **CLI** : les variables d'environnement recommandées (`GOOGLE_CLOUD_PROJECT`, `GEMINI_MODEL`, `GEMINI_SANDBOX`), les variables personnalisées, et la bascule "mode question native" — contenu repris tel quel de `AgentSettingsModal`.
- **BREAKING** (UI) : `AgentSettingsModal` et son icône engrenage dans `AgentSelector` sont supprimés ; ce contenu n'est plus accessible que via Configuration > CLI.
- L'application affiche désormais en permanence sa structure globale (barre de navigation + sidebar workspace), y compris lorsqu'aucun workspace n'est configuré, afin que Configuration reste atteignable dans tous les cas. Dans cet état, seule la zone de contenu des pages liées à un workspace (Kanban, Specs, Timeline) affiche l'invitation à ajouter un premier projet ; Agents (pool), Réglages et Configuration restent pleinement accessibles.
- La sidebar workspace (sélecteur d'agent par défaut + liste des projets) reste affichée lorsque Configuration est ouverte.
- Aucun changement backend : `GET/PATCH /api/preferences` et `GET /api/agents` couvrent déjà tout le périmètre.

## Capabilities

### New Capabilities
- `platform-configuration`: nouvelle page Configuration globale (navigation, indépendance vis-à-vis du workspace, sous-section Agents en lecture seule, sous-section CLI reprenant `AgentSettingsModal`).

### Modified Capabilities
- `agent-selection`: les requirements "Affichage adaptatif de la configuration de l'agent" et "Réglage global du mode question native", qui décrivent `AgentSettingsModal`, sont retirés/déplacés — ce comportement est désormais spécifié sous `platform-configuration`. La détection des agents, le sélecteur d'agent par défaut dans la sidebar et le verrouillage par session ne changent pas.
- `workspace-management`: le scénario "Aucun workspace configuré" change — au lieu de remplacer toute l'application par un écran d'accueil, la structure globale (navigation, sidebar) reste affichée et seule la zone de contenu invite à ajouter un premier projet.
- `agent-specialization`: retrait de la mention selon laquelle l'écran Réglages "sera conçu pour accueillir d'autres réglages de l'application par la suite" — les futurs réglages globaux de la plateforme vivent désormais dans Configuration, pas dans Réglages.

## Impact

- Frontend uniquement : `frontend/src/components/Layout.tsx` (nouvelle entrée Configuration hors liste d'onglets), `frontend/src/App.tsx` (structure toujours montée, y compris sans workspace), `frontend/src/components/AgentSelector.tsx` (suppression de l'icône engrenage et de l'usage de `AgentSettingsModal`), suppression de `frontend/src/components/AgentSettingsModal.tsx` (contenu repris dans la nouvelle page), nouvelle page `ConfigurationPage` avec ses deux sous-sections, ajustement de `frontend/src/pages/WorkspaceSetup.tsx` (devient le contenu de la zone centrale plutôt qu'un remplacement plein écran) et des locales FR/EN concernées (`navigation.json`, nouveau `configuration.json`).
- Aucun changement d'API ni de schéma de données backend.
