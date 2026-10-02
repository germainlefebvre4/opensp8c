# Benchmark plateforme vs baseline

Axe mesuré : la vitesse. Un run n'entre dans les agrégats que si le test d'acceptation caché passe.

## Configuration

- SHA de départ : `abc123`
- Agent : claude
- Modèle : opus
- Effort : high
- Relances maximum (méthode B) : 2
- Runs : méthode A (plateforme) 1, méthode B (baseline) 1

## Runs

| Run | Méthode | Validité | Temps humain | Temps machine | Total | Relances / pauses |
|---|---|---|---|---|---|---|
| a-01 | A | invalide (run de pool terminé en "paused") | 1m40s | 1m40s | 3m20s | 1 run(s) de pool |
| b-01 | B | valide | — | — | 6m40s | 0 relance(s) |

## Agrégats (runs valides uniquement)

| Méthode | Valides | Médiane | Min | Max |
|---|---|---|---|---|
| A (plateforme) | 0/1 valides | aucune donnée valide | — | — |
| B (baseline) | 1/1 valides | 6m40s | 6m40s | 6m40s |

## Écart des médianes

Aucun écart calculé : aucune donnée valide pour la méthode A.

## Limites

- Biais de familiarité : l'agent connaît ce dépôt, et la plateforme a été en partie construite avec elle-même.
- Taille d'échantillon : 1 run(s) pour A et 1 pour B ; les chiffres donnent un ordre de grandeur, pas une preuve statistique.
- Hors périmètre : la qualité du code au-delà du seuil de validité (test d'acceptation), le coût et les tokens ne sont pas mesurés.
- Le temps humain de la méthode A est une borne basse : les réponses d'explore sont scriptées, mais l'opérateur reste humain.
