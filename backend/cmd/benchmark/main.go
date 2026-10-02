// Command benchmark compares developing a feature with the platform (method A)
// against a direct agent run without it (method B).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/glefebvre/opensp8c/internal/benchmark"
)

const usage = `Usage: benchmark <command> [flags]

Commands:
  prepare   Clone the repo at the start SHA and set up a run (-method A|B -run N)
  observe   Record the platform event stream of a method-A run (-run a-01 -change <name>)
  baseline  Execute a method-B run: claude -p, validation, retries (-run b-01)
  collect   Copy raw data, run the acceptance test, write result.json (-run <id>)
  report    Aggregate every result.json into a Markdown report

Common flags: -config benchmark/benchmark.yaml  -repo .
Run "benchmark <command> -h" for the flags of a command.
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		fmt.Print(usage)
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "prepare":
		err = runPrepare(args)
	case "observe":
		err = runObserve(args)
	case "baseline":
		err = runBaseline(args)
	case "collect":
		err = runCollect(args)
	case "report":
		err = runReport(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type common struct {
	fs     *flag.FlagSet
	config *string
	repo   *string
}

func newCommon(name string) *common {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	return &common{
		fs:     fs,
		config: fs.String("config", "benchmark/benchmark.yaml", "benchmark configuration file"),
		repo:   fs.String("repo", ".", "git repository to benchmark"),
	}
}

func (c *common) load(args []string) (*benchmark.Config, error) {
	_ = c.fs.Parse(args)
	cfgPath := *c.config
	if !filepath.IsAbs(cfgPath) {
		if _, err := os.Stat(cfgPath); err != nil {
			cfgPath = filepath.Join(*c.repo, cfgPath)
		}
	}
	return benchmark.LoadConfig(cfgPath, *c.repo)
}

func runPrepare(args []string) error {
	c := newCommon("prepare")
	method := c.fs.String("method", "", "A (platform) or B (baseline); empty prepares both")
	run := c.fs.Int("run", 0, "run number (1..runs); 0 prepares every run")
	cfg, err := c.load(args)
	if err != nil {
		return err
	}
	var methods []string
	switch strings.ToUpper(*method) {
	case "":
		methods = []string{benchmark.MethodPlatform, benchmark.MethodBaseline}
	case "A", "B":
		methods = []string{strings.ToUpper(*method)}
	default:
		return fmt.Errorf("-method must be A or B")
	}
	if *run < 0 || *run > cfg.Runs {
		return fmt.Errorf("-run must be between 1 and %d (runs)", cfg.Runs)
	}
	for _, m := range methods {
		for n := 1; n <= cfg.Runs; n++ {
			if *run != 0 && n != *run {
				continue
			}
			env, err := benchmark.PrepareRun(cfg, m, n)
			if err != nil {
				return err
			}
			fmt.Printf("%s prepared\n  clone: %s\n", env.RunID, env.CloneDir)
			if m == benchmark.MethodPlatform {
				fmt.Printf("  platform config: %s (port %d, workspace id %s)\n  start the instance: %s\n  observe: benchmark observe -run %s -change <change-name>\n",
					env.ConfigYML, env.Port, env.WorkspaceID, env.StartCmd, env.RunID)
			} else {
				fmt.Printf("  execute: benchmark baseline -run %s\n", env.RunID)
			}
		}
	}
	return nil
}

func runObserve(args []string) error {
	c := newCommon("observe")
	run := c.fs.String("run", "", "run id (e.g. a-01)")
	change := c.fs.String("change", "", "name of the change to track")
	url := c.fs.String("url", "", "platform base URL (default http://127.0.0.1:<run port>)")
	cfg, err := c.load(args)
	if err != nil {
		return err
	}
	env, err := benchmark.LoadRunEnv(cfg, *run)
	if err != nil {
		return err
	}
	if env.Method != benchmark.MethodPlatform {
		return fmt.Errorf("run %s is not a method-A run", env.RunID)
	}
	if *change == "" {
		return fmt.Errorf("-change is required (the change name fixed in the brief)")
	}
	base := *url
	if base == "" {
		base = "http://127.0.0.1:" + strconv.Itoa(env.Port)
	}
	f, err := os.OpenFile(filepath.Join(env.RunDir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "observing %s (workspace %s) — Ctrl-C once the change is merged\n", base, env.WorkspaceID)
	return benchmark.Observe(ctx, benchmark.ObserverOptions{
		BaseURL: base, WorkspaceID: env.WorkspaceID, Change: *change, Out: f,
	})
}

func runBaseline(args []string) error {
	c := newCommon("baseline")
	run := c.fs.String("run", "", "run id (e.g. b-01)")
	cfg, err := c.load(args)
	if err != nil {
		return err
	}
	env, err := benchmark.LoadRunEnv(cfg, *run)
	if err != nil {
		return err
	}
	if env.Method != benchmark.MethodBaseline {
		return fmt.Errorf("run %s is not a method-B run", env.RunID)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rec, err := benchmark.RunBaseline(ctx, cfg, env, benchmark.BaselineOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("%s: %d attempt(s), %s\nnext: benchmark collect -run %s\n", env.RunID, len(rec.Attempts), rec.Status, env.RunID)
	return nil
}

func runCollect(args []string) error {
	c := newCommon("collect")
	run := c.fs.String("run", "", "run id")
	change := c.fs.String("change", "", "change name (method A; inferred when unambiguous)")
	cfg, err := c.load(args)
	if err != nil {
		return err
	}
	res, err := benchmark.Collect(context.Background(), cfg, *run, benchmark.CollectOptions{Change: *change})
	if err != nil {
		return err
	}
	verdict := "valid"
	if !res.Valid {
		verdict = "INVALID: " + strings.Join(res.InvalidReasons, "; ")
	}
	fmt.Printf("%s: %s (total %.0fs)\n", res.RunID, verdict, res.Timings.TotalSec)
	return nil
}

func runReport(args []string) error {
	c := newCommon("report")
	out := c.fs.String("out", "", "report file (default <output_dir>/report.md)")
	cfg, err := c.load(args)
	if err != nil {
		return err
	}
	results, err := benchmark.LoadResults(cfg.Resolve(cfg.OutputDir))
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no result.json under %s: run collect first", cfg.Resolve(cfg.OutputDir))
	}
	md := benchmark.RenderReport(results)
	dest := *out
	if dest == "" {
		dest = filepath.Join(cfg.Resolve(cfg.OutputDir), "report.md")
	}
	if err := os.WriteFile(dest, []byte(md), 0o644); err != nil {
		return err
	}
	fmt.Print(md)
	fmt.Fprintf(os.Stderr, "\nreport written to %s\n", dest)
	return nil
}
