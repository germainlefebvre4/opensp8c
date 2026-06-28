## 1. Capture stderr du subprocess

- [ ] 1.1 Dans `backend/internal/session/subprocess.go`, ajouter un pipe stderr sur `cmd` via `cmd.StderrPipe()`
- [ ] 1.2 Lancer une goroutine dans `StartSubprocess` qui lit le pipe stderr ligne par ligne et loggue chaque ligne avec `log.Printf("[subprocess stderr] %s", line)`
- [ ] 1.3 Vérifier que la goroutine se termine proprement à la fermeture du pipe (fin du processus)

## 2. Message d'amorce pour sessions anonymes

- [ ] 2.1 Dans `backend/internal/session/manager.go`, dans `StartAnonymous`, injecter un message d'amorce sur stdin du subprocess après `m.startFanOut(...)` :
  ```json
  {"type":"user","message":{"role":"user","content":"Présente-toi en une phrase courte et invite l'utilisateur à décrire ce qu'il veut explorer ou construire."}}
  ```
- [ ] 2.2 S'assurer que le message est suivi d'un `\n` (même pattern que l'injection de la session nommée)

## 3. Vérification

- [ ] 3.1 Démarrer le backend, cliquer sur "+" dans la colonne "To Explore", vérifier que le panel reste "connecté" et qu'un message de bienvenue apparaît
- [ ] 3.2 Vérifier dans les logs backend qu'aucun stderr du subprocess n'est émis (session saine)
- [ ] 3.3 Envoyer un message depuis le panel et vérifier que la réponse est reçue
