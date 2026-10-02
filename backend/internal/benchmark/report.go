package benchmark

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Median returns the median of xs (mean of the two middle values when even).
func Median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func fmtDur(sec float64) string {
	total := int(math.Round(sec))
	h, m, s := total/3600, (total%3600)/60, total%60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

type aggregate struct {
	valid, total   int
	totals         []float64
	humans, machs  []float64
	runsOfThisKind []*Result
}

func aggregateMethod(results []*Result, method string) aggregate {
	var a aggregate
	for _, r := range results {
		if r.Method != method {
			continue
		}
		a.total++
		a.runsOfThisKind = append(a.runsOfThisKind, r)
		if !r.Valid {
			continue
		}
		a.valid++
		a.totals = append(a.totals, r.Timings.TotalSec)
		if method == MethodPlatform {
			a.humans = append(a.humans, r.Timings.HumanSec)
			a.machs = append(a.machs, r.Timings.MachineSec)
		}
	}
	return a
}

func minMax(xs []float64) (float64, float64) {
	lo, hi := xs[0], xs[0]
	for _, x := range xs {
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	return lo, hi
}

// RenderReport builds the Markdown report from raw results only. It is pure:
// no file, network or platform access.
func RenderReport(results []*Result) string {
	sorted := append([]*Result(nil), results...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Method != sorted[j].Method {
			return sorted[i].Method < sorted[j].Method
		}
		return sorted[i].RunID < sorted[j].RunID
	})
	a := aggregateMethod(sorted, MethodPlatform)
	b := aggregateMethod(sorted, MethodBaseline)

	var w strings.Builder
	w.WriteString("# Benchmark plateforme vs baseline\n\n")
	w.WriteString("Axe mesuré : la vitesse. Un run n'entre dans les agrégats que si le test d'acceptation caché passe.\n\n")

	w.WriteString("## Configuration\n\n")
	if len(sorted) > 0 {
		c := sorted[0]
		fmt.Fprintf(&w, "- SHA de départ : `%s`\n", c.StartSHA)
		fmt.Fprintf(&w, "- Agent : %s\n- Modèle : %s\n- Effort : %s\n", orDash(c.Config.Agent), orDash(c.Config.Model), orDash(c.Config.Effort))
		fmt.Fprintf(&w, "- Relances maximum (méthode B) : %d\n", c.Config.MaxRetries)
	}
	fmt.Fprintf(&w, "- Runs : méthode A (plateforme) %d, méthode B (baseline) %d\n\n", a.total, b.total)

	w.WriteString("## Runs\n\n")
	w.WriteString("| Run | Méthode | Validité | Temps humain | Temps machine | Total | Relances / pauses |\n")
	w.WriteString("|---|---|---|---|---|---|---|\n")
	for _, r := range sorted {
		validity := "valide"
		if !r.Valid {
			validity = "invalide"
			if len(r.InvalidReasons) > 0 {
				validity += " (" + strings.Join(r.InvalidReasons, " ; ") + ")"
			}
		}
		human, machine, extra := "—", "—", ""
		if r.Method == MethodPlatform {
			if r.Timings.TotalSec > 0 {
				human, machine = fmtDur(r.Timings.HumanSec), fmtDur(r.Timings.MachineSec)
			}
			extra = fmt.Sprintf("%d run(s) de pool", r.Timings.PoolRuns)
			if r.PauseReason != "" {
				extra += ", pause : " + r.PauseReason
			}
		} else {
			extra = fmt.Sprintf("%d relance(s)", r.Retries)
		}
		total := "—"
		if r.Timings.TotalSec > 0 {
			total = fmtDur(r.Timings.TotalSec)
		}
		fmt.Fprintf(&w, "| %s | %s | %s | %s | %s | %s | %s |\n", r.RunID, r.Method, validity, human, machine, total, extra)
	}
	w.WriteString("\n")

	w.WriteString("## Agrégats (runs valides uniquement)\n\n")
	w.WriteString("| Méthode | Valides | Médiane | Min | Max |\n|---|---|---|---|---|\n")
	for _, m := range []struct {
		label string
		agg   aggregate
	}{{"A (plateforme)", a}, {"B (baseline)", b}} {
		if m.agg.valid == 0 {
			fmt.Fprintf(&w, "| %s | %d/%d valides | aucune donnée valide | — | — |\n", m.label, m.agg.valid, m.agg.total)
			continue
		}
		lo, hi := minMax(m.agg.totals)
		fmt.Fprintf(&w, "| %s | %d/%d valides | %s | %s | %s |\n", m.label, m.agg.valid, m.agg.total, fmtDur(Median(m.agg.totals)), fmtDur(lo), fmtDur(hi))
	}
	w.WriteString("\n")
	if a.valid > 0 {
		fmt.Fprintf(&w, "Méthode A, médiane par composante : temps humain %s, temps machine %s.\n\n",
			fmtDur(Median(a.humans)), fmtDur(Median(a.machs)))
	}

	w.WriteString("## Écart des médianes\n\n")
	switch {
	case a.valid == 0 && b.valid == 0:
		w.WriteString("Aucun écart calculé : aucune donnée valide pour les deux méthodes.\n\n")
	case a.valid == 0:
		w.WriteString("Aucun écart calculé : aucune donnée valide pour la méthode A.\n\n")
	case b.valid == 0:
		w.WriteString("Aucun écart calculé : aucune donnée valide pour la méthode B.\n\n")
	default:
		ma, mb := Median(a.totals), Median(b.totals)
		diff := ma - mb
		sign, who := "+", "plus lente"
		if diff < 0 {
			sign, who = "-", "plus rapide"
		}
		fmt.Fprintf(&w, "Médiane A − médiane B = %s%s : la plateforme est %s que la baseline", sign, fmtDur(math.Abs(diff)), who)
		if mb > 0 {
			fmt.Fprintf(&w, " (%.0f %% de la médiane de la baseline)", math.Abs(diff)/mb*100)
		}
		w.WriteString(".\n\n")
	}

	w.WriteString("## Limites\n\n")
	w.WriteString("- Biais de familiarité : l'agent connaît ce dépôt, et la plateforme a été en partie construite avec elle-même.\n")
	fmt.Fprintf(&w, "- Taille d'échantillon : %d run(s) pour A et %d pour B ; les chiffres donnent un ordre de grandeur, pas une preuve statistique.\n", a.total, b.total)
	w.WriteString("- Hors périmètre : la qualité du code au-delà du seuil de validité (test d'acceptation), le coût et les tokens ne sont pas mesurés.\n")
	w.WriteString("- Le temps humain de la méthode A est une borne basse : les réponses d'explore sont scriptées, mais l'opérateur reste humain.\n")
	return w.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
