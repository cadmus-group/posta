package openapiagg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type SourceStatus struct {
	Name            string    `json:"name"`
	OpenAPIURL      string    `json:"openapiUrl"`
	PathPrefix      string    `json:"pathPrefix"`
	ComponentPrefix string    `json:"componentPrefix"`
	OperationPrefix string    `json:"operationPrefix"`
	LastAttempt     time.Time `json:"lastAttempt"`
	LastSuccess     time.Time `json:"lastSuccess"`
	Error           string    `json:"error,omitempty"`
	Issues          []string  `json:"issues,omitempty"`
}

type state struct {
	doc           *openapi3.T
	docJSON       []byte
	mergeIssues   []string
	statuses      []SourceStatus
	lastErr       error
	lastRefresh   time.Time
	lastSuccess   time.Time
	refreshPeriod time.Duration
}

// Server manages fetching, merging, and serving OpenAPI docs.
type Server struct {
	log             *slog.Logger
	cfgPath         string
	refreshOverride time.Duration
	client          *http.Client

	mu    sync.RWMutex
	state state
}

func NewServer(cfgPath string, refreshOverride time.Duration, client *http.Client, log *slog.Logger) *Server {
	if log == nil {
		log = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if client == nil {
		client = newHTTPClient()
	}
	return &Server{
		log:             log,
		cfgPath:         cfgPath,
		refreshOverride: refreshOverride,
		client:          client,
		state: state{
			refreshPeriod: time.Duration(defaultRefreshSeconds) * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context, addr string) error {
	if err := s.refresh(ctx); err != nil {
		s.log.Warn("initial refresh failed", "err", err)
	}

	go s.refreshLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("/sources", s.handleSources)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	s.log.Info("openapi-aggregator listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.getRefreshPeriod())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refresh(ctx); err != nil {
				s.log.Warn("refresh failed", "err", err)
			}
			ticker.Reset(s.getRefreshPeriod())
		}
	}
}

func (s *Server) refresh(ctx context.Context) error {
	cfg, err := LoadConfig(s.cfgPath)
	if err != nil {
		s.setFailure(nil, nil, fmt.Errorf("config: %w", err))
		return err
	}

	refreshEvery := time.Duration(cfg.RefreshSeconds) * time.Second
	if s.refreshOverride > 0 {
		refreshEvery = s.refreshOverride
	}
	s.setRefreshPeriod(refreshEvery)

	var (
		statuses []SourceStatus
		success  []mergeInput
		hadFail  bool
	)

	for _, src := range cfg.Sources {
		status := SourceStatus{
			Name:            src.Name,
			OpenAPIURL:      src.OpenAPIURL,
			PathPrefix:      normalizePrefix(src.PathPrefix),
			ComponentPrefix: src.ComponentPrefixSafe(),
			OperationPrefix: src.OperationPrefixSafe(),
			LastAttempt:     time.Now().UTC(),
		}

		ctxFetch, cancel := context.WithTimeout(ctx, defaultHTTPDelay+2*time.Second)
		spec, err := fetchSpec(ctxFetch, s.client, src)
		cancel()

		if err != nil {
			status.Error = err.Error()
			statuses = append(statuses, status)
			hadFail = true
			continue
		}

		status.LastSuccess = status.LastAttempt
		success = append(success, mergeInput{Source: src, Doc: spec.Doc})
		statuses = append(statuses, status)
	}

	if len(success) == 0 {
		err := errors.New("no specs fetched")
		s.setFailure(statuses, nil, err)
		return err
	}

	merged, mergeIssues, err := mergeSpecs(success)
	if err != nil {
		s.setFailure(statuses, mergeIssuesToStrings(mergeIssues), err)
		return err
	}

	issueStrings := mergeIssuesToStrings(mergeIssues)
	var refreshErr error
	if hadFail {
		refreshErr = errors.New("one or more sources failed")
	}
	if len(issueStrings) > 0 {
		if refreshErr != nil {
			refreshErr = errors.Join(refreshErr, fmt.Errorf("merged with %d issues", len(issueStrings)))
		} else {
			refreshErr = fmt.Errorf("merged with %d issues", len(issueStrings))
		}
	}
	s.setSuccess(statuses, issueStrings, merged, refreshErr)

	if refreshErr != nil {
		return refreshErr
	}
	return nil
}

func (s *Server) setFailure(statuses []SourceStatus, issues []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if statuses == nil {
		statuses = s.state.statuses
	}
	s.state.lastErr = err
	s.state.lastRefresh = now
	s.state.mergeIssues = issues
	s.state.statuses = statuses
}

func (s *Server) setSuccess(statuses []SourceStatus, issues []string, doc *openapi3.T, refreshErr error) {
	data, err := json.Marshal(doc)
	if err != nil {
		refreshErr = fmt.Errorf("marshal merged spec: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.state.doc = doc
	if err == nil {
		s.state.docJSON = data
	}
	s.state.mergeIssues = issues
	s.state.statuses = statuses
	s.state.lastRefresh = now
	if refreshErr == nil {
		s.state.lastSuccess = now
	}
	s.state.lastErr = refreshErr
}

func (s *Server) getRefreshPeriod() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.refreshPeriod
}

func (s *Server) setRefreshPeriod(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.refreshPeriod = d
}

func (s *Server) snapshot() state {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	snap := s.snapshot()
	if snap.docJSON == nil || snap.lastErr != nil {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	snap := s.snapshot()
	if snap.docJSON == nil {
		http.Error(w, "no merged spec available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(snap.docJSON)
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	snap := s.snapshot()
	resp := map[string]any{
		"sources":     snap.statuses,
		"mergeIssues": snap.mergeIssues,
		"lastRefresh": snap.lastRefresh,
		"lastSuccess": snap.lastSuccess,
		"ready":       snap.docJSON != nil && snap.lastErr == nil,
	}
	if snap.lastErr != nil {
		resp["error"] = snap.lastErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func mergeIssuesToStrings(issues []mergeIssue) []string {
	out := make([]string, 0, len(issues))
	for _, i := range issues {
		out = append(out, fmt.Sprintf("%s: %s", i.Source, i.Detail))
	}
	return out
}
