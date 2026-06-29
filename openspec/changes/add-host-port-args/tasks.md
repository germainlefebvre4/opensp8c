## 1. Parsing des arguments CLI

- [x] 1.1 Ajouter l'import du package `flag` dans `backend/cmd/server/main.go`
- [x] 1.2 Déclarer les flags `--host` (défaut `""`) et `--port` (défaut `""`) avant `flag.Parse()`
- [x] 1.3 Appeler `flag.Parse()` en début de `main()`

## 2. Résolution de la priorité host/port

- [x] 2.1 Implémenter la résolution du port : flag `--port` > env `PORT` > défaut `"8080"`
- [x] 2.2 Implémenter la résolution du host : flag `--host` > env `HOST` > défaut `"0.0.0.0"`

## 3. Mise à jour du serveur

- [x] 3.1 Mettre à jour `srv.Addr` pour utiliser `host + ":" + port` au lieu de `":" + port`
- [x] 3.2 Mettre à jour le log de démarrage pour afficher `host:port` complet
