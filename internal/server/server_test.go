package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/cansarihan/culprit/internal/demo"
	"github.com/cansarihan/culprit/internal/scan"
)

func demoServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if err := demo.Build(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(Options{Repo: dir, Token: token}, BuildInfo{Version: "test"}).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func getJSON(t *testing.T, url string, v any) int {
	t.Helper()
	res, err := http.Get(url) // #nosec G107 -- url is the httptest server under test
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if v != nil {
		_ = json.NewDecoder(res.Body).Decode(v)
	}
	return res.StatusCode
}

func TestReportAndCommit(t *testing.T) {
	srv := demoServer(t, "")
	var rep scan.Report
	if code := getJSON(t, srv.URL+"/api/v1/report", &rep); code != http.StatusOK {
		t.Fatalf("report = %d", code)
	}
	if rep.Silent != 4 || len(rep.Assessments) != demo.Commits() {
		t.Fatalf("report silent=%d commits=%d", rep.Silent, len(rep.Assessments))
	}
	top := rep.Assessments[0].Commit.Hash
	var detail CommitDetail
	if code := getJSON(t, srv.URL+"/api/v1/commits/"+top[:10], &detail); code != http.StatusOK {
		t.Fatalf("commit = %d", code)
	}
	if detail.Assessment.Commit.Hash != top || len(detail.Files) == 0 {
		t.Errorf("detail = %+v", detail.Assessment.Commit)
	}
}

func TestCommitValidation(t *testing.T) {
	srv := demoServer(t, "")
	if code := getJSON(t, srv.URL+"/api/v1/commits/--output=x", nil); code != http.StatusBadRequest {
		t.Errorf("option-looking ref = %d, want 400", code)
	}
	if code := getJSON(t, srv.URL+"/api/v1/commits/deadbeef", nil); code != http.StatusNotFound {
		t.Errorf("unknown commit = %d, want 404", code)
	}
}

func TestSignalsAndToken(t *testing.T) {
	srv := demoServer(t, "secret")
	if code := getJSON(t, srv.URL+"/api/v1/signals", nil); code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", code)
	}
	var catalog []map[string]any
	if code := getJSON(t, srv.URL+"/api/v1/signals?token=secret", &catalog); code != http.StatusOK || len(catalog) < 15 {
		t.Errorf("signals = %d with %d entries", code, len(catalog))
	}
}

func TestPanelIsNotCached(t *testing.T) {
	srv := httptest.NewServer(New(Options{Repo: t.TempDir(), UI: true}, BuildInfo{}).Handler())
	defer srv.Close()
	res, err := http.Get(srv.URL + "/app.js")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("app.js: status %d, cache-control %q", res.StatusCode, res.Header.Get("Cache-Control"))
	}
}
