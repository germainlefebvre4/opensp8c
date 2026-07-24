package api

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/glefebvre/opensp8c/internal/api/handlers"
	"github.com/glefebvre/opensp8c/internal/config"
	"github.com/glefebvre/opensp8c/internal/conversation"
	"github.com/glefebvre/opensp8c/internal/preferences"
	"github.com/glefebvre/opensp8c/internal/session"
	"github.com/glefebvre/opensp8c/internal/watcher"
	"github.com/glefebvre/opensp8c/internal/workspace"
	"github.com/glefebvre/opensp8c/ui"
)

func preferencesPath(cfgPath string) string {
	if p := os.Getenv("PREFERENCES_PATH"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(cfgPath), "preferences.json")
}

func conversationsPath(cfgPath string) string {
	if p := os.Getenv("CONVERSATIONS_PATH"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(cfgPath), "conversations")
}

func draftsPath(cfgPath string) string {
	if p := os.Getenv("DRAFTS_PATH"); p != "" {
		return p
	}
	return filepath.Join(filepath.Dir(cfgPath), "drafts")
}

func NewRouter(cfg *config.Config, cfgPath string) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	prefsSvc := preferences.NewService(preferencesPath(cfgPath))
	convStore := conversation.NewStore(conversationsPath(cfgPath))
	mgr := session.NewManager(prefsSvc, convStore)

	watcherSvc := watcher.NewWatcherService()
	for _, wc := range cfg.Workspaces {
		absPath, _ := filepath.Abs(wc.Path)
		_ = watcherSvc.StartWatching(workspace.StableID(absPath), absPath)
	}

	go conversation.StartRetentionLoop(cfg, prefsSvc, convStore, time.Hour)

	wsHandler := handlers.NewWorkspaceHandler(cfg, cfgPath)
	kanbanHandler := handlers.NewKanbanHandler(wsHandler, prefsSvc)
	specsHandler := handlers.NewSpecsHandler(wsHandler)
	archiveHandler := handlers.NewArchiveHandler(wsHandler)
	tagsHandler := handlers.NewTagsHandler(wsHandler)
	taskHandler := handlers.NewTaskHandler(wsHandler)
	ffHandler := handlers.NewFFHandler(wsHandler, mgr, convStore, watcherSvc)
	exploreHandler := handlers.NewExploreHandler(wsHandler, mgr, prefsSvc, watcherSvc, convStore, draftsPath(cfgPath))
	eventsHandler := handlers.NewEventsHandler(wsHandler, watcherSvc)
	prefsHandler := handlers.NewPreferencesHandler(prefsSvc)

	r.Route("/api", func(r chi.Router) {
		r.Use(jsonContentType)

		r.Get("/workspaces", wsHandler.List)
		r.Post("/workspaces", wsHandler.Add)
		r.Delete("/workspaces/{id}", wsHandler.Delete)

		r.Get("/workspaces/{id}/changes", kanbanHandler.ListChanges)
		r.Get("/workspaces/{id}/changes/{name}", kanbanHandler.GetChange)
		r.Get("/workspaces/{id}/archived-changes", kanbanHandler.ListArchivedChanges)

		r.Get("/workspaces/{id}/specs", specsHandler.ListSpecs)
		r.Get("/workspaces/{id}/specs/overview", specsHandler.GetOverview)
		r.Get("/workspaces/{id}/specs/{name}", specsHandler.GetSpec)
		r.Put("/workspaces/{id}/specs/{name}", specsHandler.UpdateSpec)

		r.Post("/workspaces/{id}/changes/{name}/archive", archiveHandler.Archive)
		r.Post("/workspaces/{id}/changes/{name}/retag", tagsHandler.Retag)
		r.Patch("/workspaces/{id}/changes/{name}/tasks/reset", ffHandler.ResetTasks)
		r.Patch("/workspaces/{id}/changes/{name}/tasks/{index}", taskHandler.PatchTask)

		r.Post("/workspaces/{id}/changes/{name}/ff", ffHandler.TriggerFF)
		r.Get("/workspaces/{id}/changes/{name}/conversations/{kind}", ffHandler.ListConversationRuns)
		r.Get("/workspaces/{id}/changes/{name}/conversations/{kind}/{ts}", ffHandler.GetConversationRun)

		r.Get("/workspaces/{id}/changes/{name}/explore", exploreHandler.HandleWS)
		r.Delete("/workspaces/{id}/changes/{name}/explore", exploreHandler.StopSession)

		r.Post("/workspaces/{id}/explore/sessions", exploreHandler.CreateAnonymousSession)
		r.Get("/workspaces/{id}/explore/sessions/{sessionId}", exploreHandler.HandleAnonymousWS)
		r.Delete("/workspaces/{id}/explore/sessions/{sessionId}", exploreHandler.StopAnonymousSession)

		r.Post("/workspaces/{id}/explorations/{ghostId}/promote", exploreHandler.PromoteGhost)
		r.Delete("/workspaces/{id}/explorations/{ghostId}", exploreHandler.DeleteGhost)
		r.Get("/workspaces/{id}/explorations/{ghostId}/draft", exploreHandler.GetGhostDraft)
		r.Put("/workspaces/{id}/explorations/{ghostId}/draft", exploreHandler.UpdateGhostDraft)
		r.Delete("/workspaces/{id}/explorations/{ghostId}/draft", exploreHandler.DeleteGhostDraft)

		r.Get("/workspaces/{id}/events", eventsHandler.HandleSSE)

		r.Get("/agents", prefsHandler.ListAgents)
		r.Get("/preferences", prefsHandler.GetPreferences)
		r.Patch("/preferences", prefsHandler.PatchPreferences)
	})

	r.Handle("/*", staticHandler(ui.FS()))

	return r
}

// staticHandler serves a SPA: exact files first, index.html fallback for unknown paths.
func staticHandler(distFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(distFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			serveIndex(w, distFS)
			return
		}
		// Strip leading slash for fs.Open
		f, err := distFS.Open(path[1:])
		if err != nil {
			serveIndex(w, distFS)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, distFS fs.FS) {
	f, err := distFS.Open("index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.Copy(w, f)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
