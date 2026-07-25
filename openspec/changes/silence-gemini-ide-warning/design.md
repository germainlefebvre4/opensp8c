## Context

Lors de l'analyse d'erreurs stderr sur le subprocess Gemini, l'erreur `Failed to connect to IDE companion extension` est émise de manière continue si l'extension compagnon n'est pas lancée. Actuellement, le backend scanne stderr et transmet cette erreur au frontend sous forme de `session_warning` non fatal. Bien que non fatal, ce warning génère de la confusion chez l'utilisateur et surcharge inutilement les fichiers de log de session.

## Goals / Non-Goals

**Goals:**
- Ignorer silencieusement toute ligne stderr contenant `Failed to connect to IDE companion extension`.
- Éviter d'enregistrer cette erreur dans `SessionLog` ou de l'afficher dans la console du serveur.
- Ne plus générer l'événement `session_warning` pour cette erreur.
- Mettre à jour le test unitaire `TestStartSubprocessGeminiBridge_ThrottledIDEWarning` devenu `TestStartSubprocessGeminiBridge_SilencedIDEWarning` pour valider que l'erreur est bien ignorée.

**Non-Goals:**
- Supprimer la prise en charge d'autres `session_warning` légitimes (comme `TerminalQuotaError` ou `ProjectIdRequiredError`).
- Intercepter d'autres flux d'erreurs sans rapport.

## Decisions

### Décision 1 : Intercepter et filtrer l'erreur directement dans la boucle de scan stderr
- **Option A (Masquage UI uniquement) :** Conserver les logs console du serveur et masquer uniquement l'événement `session_warning` envoyé au client.
- **Option B (Silence Absolu - Sélectionnée) :** Filtrer la ligne contenant l'erreur dès sa détection dans le scanner, avant même de faire `log.Printf` ou de l'ajouter au `sessionLog`.
- **Raisonnement :** L'option B est préférable car l'erreur d'extension compagnon n'apporte aucune valeur ajoutée au niveau des logs backend ou des fichiers de session si l'utilisateur choisit consciemment de travailler sans l'extension. Elle est donc filtrée à la source.

## Risks / Trade-offs

- **[Risque]** Un utilisateur pourrait rencontrer des problèmes légitimes avec son extension compagnon sans que l'erreur n'apparaisse dans les logs pour le diagnostic.
  - **Mitigation :** L'utilisation de l'extension compagnon est optionnelle pour le fonctionnement de base de l'exploration de code. Si un diagnostic s'avère nécessaire, l'utilisateur pourra exécuter manuellement l'agent en mode verbeux dans son terminal.
