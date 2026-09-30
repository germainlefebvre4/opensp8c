# Tasks

## 1. Cadencer le producteur du test

- [ ] 1.1 Dans `TestServeWSDeliversLiveMessagesPastBufferCapacity` (`backend/internal/api/handlers/explore_test.go`), remplacer la pause `time.Sleep` par un canal de crédits de capacité 300 : le producteur prend un crédit avant chaque injection, le lecteur en rend un après chaque événement `assistant` vérifié. Vérifier que la constante est commentée (fenêtre de 500 entrées, marge de 200) et que `go test -run TestServeWSDeliversLiveMessagesPastBufferCapacity ./internal/api/handlers/` passe.
- [ ] 1.2 Ajouter un canal `done` fermé en `defer`, sur lequel le producteur fait `select` avec l'obtention d'un crédit. Vérifier qu'une sortie anticipée du test (par exemple en provoquant temporairement un `t.Fatalf` dans la boucle de lecture) ne laisse aucune goroutine bloquée, puis retirer ce `t.Fatalf`.
- [ ] 1.3 Injecter le message `result` final sans crédit, et réécrire le commentaire du test : intention « aucune perte quand le client suit, au-delà de la fenêtre », renvoi au scénario « Livraison en direct après saturation du buffer » du spec `explore-session`, et mention que le client trop lent n'est pas couvert. Vérifier la relecture du commentaire et que `go vet ./internal/api/handlers/` est propre.

## 2. Vérification de la stabilité

- [ ] 2.1 Lancer le test seul 20 fois de suite avec `go test -race -count=20 -run TestServeWSDeliversLiveMessagesPastBufferCapacity ./internal/api/handlers/` et vérifier 20 réussites.
- [ ] 2.2 Reproduire la charge qui faisait échouer le test 5 fois sur 5 (binaire de test compilé avec `-race`, 2 boucles actives par cœur pendant l'exécution) et vérifier 5 réussites sur 5.
- [ ] 2.3 Lancer `go test -race ./...` dans `backend` au moins 3 fois et vérifier que `TestServeWSDeliversLiveMessagesPastBufferCapacity` n'échoue plus dans aucune de ces exécutions.
