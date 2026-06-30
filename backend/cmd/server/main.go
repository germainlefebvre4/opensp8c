package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/glefebvre/opensp8c/internal/api"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/openspec"
)

func main() {
	hostFlag := flag.String("host", "", "")
	portFlag := flag.String("port", "", "")
	flag.Parse()

	cfgPath := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	host := *hostFlag
	if host == "" {
		host = os.Getenv("HOST")
	}
	if host == "" {
		host = "0.0.0.0"
	}

	router := api.NewRouter(cfg, cfgPath)

	// Background batch: tag all untagged changes in each workspace
	go func() {
		for _, wc := range cfg.Workspaces {
			absPath, err := filepath.Abs(wc.Path)
			if err != nil {
				continue
			}
			tagUntaggedChanges(absPath)
		}
	}()

	srv := &http.Server{
		Addr:    host + ":" + port,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s:%s", host, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func tagUntaggedChanges(workspacePath string) {
	changesDir := filepath.Join(workspacePath, "openspec", "changes")

	type changeEntry struct {
		root    string
		created string
	}

	var entries []changeEntry

	collectDir := func(dir string) {
		dirs, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range dirs {
			if !e.IsDir() || e.Name() == "archive" {
				continue
			}
			metaPath := filepath.Join(dir, e.Name(), ".openspec.yaml")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var metaWrapper struct {
				Tags    *openspec.Tags `yaml:"tags"`
				Created string         `yaml:"created"`
			}
			if yaml.Unmarshal(data, &metaWrapper) == nil && metaWrapper.Tags != nil {
				continue
			}
			created := metaWrapper.Created
			entries = append(entries, changeEntry{
				root:    filepath.Join(dir, e.Name()),
				created: created,
			})
		}
	}

	collectDir(changesDir)
	collectDir(filepath.Join(changesDir, "archive"))

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].created < entries[j].created
	})

	for _, e := range entries {
		_ = openspec.TagChange(e.root, workspacePath, false)
	}
}
