# Benchmark plateforme vs baseline

Axe mesuré : la vitesse. Un run n'entre dans les agrégats que si le test d'acceptation caché passe.

## Configuration

- SHA de départ : `abc123`
- Agent : claude
- Modèle : opus
- Effort : high
- Relances maximum (méthode B) : 2
- Runs : méthode A (plateforme) 3, méthode B (baseline) 3

## Runs

| Run | Méthode | Validité | Temps humain | Temps machine | Total | Relances / pauses |
|---|---|---|---|---|---|---|
| a-01 | A | valide | 2m00s | 10m00s | 12m00s | 1 run(s) de pool |
| a-02 | A | invalide (test d'acceptation en échec) | 1m40s | 15m00s | 16m40s | 1 run(s) de pool |
| a-03 | A | valide | 2m20s | 11m40s | 14m00s | 1 run(s) de pool |
| b-01 | B | valide | — | — | 8m20s | 0 relance(s) |
| b-02 | B | valide | — | — | 15m00s | 1 relance(s) |
| b-03 | B | valide | — | — | 11m40s | 0 relance(s) |

## Agrégats (runs valides uniquement)

| Méthode | Valides | Médiane | Min | Max |
|---|---|---|---|---|
| A (plateforme) | 2/3 valides | 13m00s | 12m00s | 14m00s |
| B (baseline) | 3/3 valides | 11m40s | 8m20s | 15m00s |

Méthode A, médiane par composante : temps humain 2m10s, temps machine 10m50s.

## Écart des médianes

Médiane A − médiane B = +1m20s : la plateforme est plus lente que la baseline (11 % de la médiane de la baseline).

## Limites

- Biais de familiarité : l'agent connaît ce dépôt, et la plateforme a été en partie construite avec elle-même.
- Taille d'échantillon : 3 run(s) pour A et 3 pour B ; les chiffres donnent un ordre de grandeur, pas une preuve statistique.
- Hors périmètre : la qualité du code au-delà du seuil de validité (test d'acceptation), le coût et les tokens ne sont pas mesurés.
- Le temps humain de la méthode A est une borne basse : les réponses d'explore sont scriptées, mais l'opérateur reste humain.
