## Context

Le serveur Go (`backend/cmd/server/main.go`) lit uniquement la variable d'environnement `PORT` pour configurer son port d'écoute et bind toujours sur toutes les interfaces (`0.0.0.0`). Il n'y a pas de parsing d'arguments CLI. La modification est localisée à ce seul fichier.

## Goals / Non-Goals

**Goals:**
- Ajouter les flags `--host` et `--port` via le package `flag` de la stdlib Go
- Respecter la chaîne de priorité : CLI arg > env var > défaut
- Mettre à jour le log de démarrage pour afficher `host:port`

**Non-Goals:**
- Ajouter une dépendance externe (pas de `cobra`, `urfave/cli`, etc.)
- Configurer le frontend Vite (port et host du dev server sont hors scope)
- Valider le format de l'adresse IP (laisser Go échouer naturellement au `ListenAndServe`)

## Decisions

### Package `flag` de la stdlib plutôt qu'une lib externe

Le parsing nécessaire est minimal (deux flags simples). Le package `flag` couvre exactement ce besoin sans introduire de dépendance. `cobra` ou `urfave/cli` apporteraient de la complexité (sous-commandes, aide auto-formatée) inutile ici.

### `--host` défaut `0.0.0.0`, pas `""`

Utiliser `""` comme défaut interne (valeur sentinelle "non fourni") permet de distinguer "l'utilisateur n'a pas passé de flag" de "l'utilisateur a explicitement passé `--host 0.0.0.0`". La résolution vers `0.0.0.0` se fait après la chaîne de priorité.

```
flag non fourni → vérifier HOST env → si vide → "0.0.0.0"
flag fourni     → utiliser la valeur du flag
```

### Adresse d'écoute : `host:port` au lieu de `:port`

Go's `net.Listen` accepte `host:port`. Avec `host = "0.0.0.0"`, le comportement est identique à l'actuel `":port"`. La migration est transparente.

## Risks / Trade-offs

- [Compatibilité] Les scripts qui passaient `PORT=X` continuent de fonctionner. Les scripts qui passaient `HOST=X` (variable custom) doivent migrer vers `--host X` ou la var `HOST`. → Risque faible car la var `HOST` n'était pas documentée.
- [Validation] Aucune validation du format host/port n'est ajoutée. Une valeur invalide produit une erreur au démarrage via `ListenAndServe`. → Comportement acceptable et cohérent avec Go idiomatique.
