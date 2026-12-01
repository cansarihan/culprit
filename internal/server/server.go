// Package server wraps the history scan in an HTTP API and serves the embedded
// panel. It reads one fixed repository chosen at startup, so a request can never
// point it at another path on the host.
package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/patch"
	"github.com/cansarihan/culprit/internal/scan"
	"github.com/cansarihan/culprit/internal/score"
	"github.com/cansarihan/culprit/internal/signals"
)

// BuildInfo is the version metadata reported on the status endpoint.
type BuildInfo struct {
	Version   string
	Commit    string
	GoVersion string
	Platform  string
}

// Options configures a server instance.
type Options struct {
	Repo  string
	Log   gitlog.Options
	Token string
	UI    bool
}

// Server caches the latest scan of one repository.
type Server struct {
	options Options
	info    BuildInfo
	started time.Time

	mu      sync.Mutex
	report  *scan.Report
	scanned time.Time
}

// New builds a server over one repository.
func New(options Options, info BuildInfo) *Server {
	return &Server{options: options, info: info, started: time.Now()}
}

// Handler returns the HTTP handler for the API and, when enabled, the panel.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/v1/status", s.guard(s.handleStatus))
	mux.HandleFunc("GET /api/v1/report", s.guard(s.handleReport))
	mux.HandleFunc("GET /api/v1/commits/{hash}", s.guard(s.handleCommit))
	mux.HandleFunc("GET /api/v1/signals", s.guard(s.handleSignals))
	if s.options.UI {
		mux.Handle("GET /", uiHandler())
	}
	return mux
}

func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	token := s.options.Token
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if provided == "" {
				provided = r.URL.Query().Get("token")
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next(w, r)
	}
}

// current returns the cached report, scanning when there is none or a refresh
// was asked for. Scans are serialized so concurrent requests share one.
func (s *Server) current(ctx context.Context, refresh bool) (*scan.Report, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.report != nil && !refresh {
		return s.report, s.scanned, nil
	}
	r, err := scan.Run(ctx, s.options.Repo, s.options.Log, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	s.report, s.scanned = &r, time.Now()
	return s.report, s.scanned, nil
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	rep, scanned, err := s.current(r.Context(), false)
	status := map[string]any{
		"version":    s.info.Version,
		"commit":     s.info.Commit,
		"go_version": s.info.GoVersion,
		"platform":   s.info.Platform,
		"repo":       s.options.Repo,
		"uptime":     time.Since(s.started).Round(time.Second).String(),
	}
	if err != nil {
		status["error"] = err.Error()
	} else {
		status["scanned"] = rep.Scanned
		status["silent"] = rep.Silent
		status["scanned_at"] = scanned
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	rep, _, err := s.current(r.Context(), r.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

var hashPattern = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)

// CommitDetail is one commit's verdict together with its diff.
type CommitDetail struct {
	Assessment score.Assessment `json:"assessment"`
	Files      []patch.FileDiff `json:"files"`
}

func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !hashPattern.MatchString(hash) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "not a commit hash"})
		return
	}
	commits, err := gitlog.Log(r.Context(), s.options.Repo, gitlog.Options{Limit: 1, Rev: hash})
	if err != nil || len(commits) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "commit not found"})
		return
	}
	files, err := patch.Show(r.Context(), s.options.Repo, commits[0].Hash)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, CommitDetail{Assessment: score.Assess(commits[0], files), Files: files})
}

func (s *Server) handleSignals(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, signals.Catalog())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
