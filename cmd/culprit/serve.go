package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/cansarihan/culprit/internal/gitlog"
	"github.com/cansarihan/culprit/internal/logging"
	"github.com/cansarihan/culprit/internal/server"
)

func runServe(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := flags.String("listen", "127.0.0.1:8080", "listen address")
	repo := flags.String("repo", ".", "repository to analyze")
	since := flags.String("since", "", "only commits newer than this")
	limit := flags.Int("n", 0, "only the most recent N commits")
	token := flags.String("token", os.Getenv("CULPRIT_TOKEN"), "require this bearer token")
	noUI := flags.Bool("no-ui", false, "serve the API without the panel")
	if err := flags.Parse(args); err != nil {
		return err
	}
	logger := logging.New("info", "text", os.Stderr)
	ctx, stop := signalContext()
	defer stop()

	handler := server.New(server.Options{
		Repo: *repo, Log: gitlog.Options{Since: *since, Limit: *limit}, Token: *token, UI: !*noUI,
	}, server.BuildInfo{Version: version, Commit: commit, GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH}).Handler()

	httpServer := &http.Server{Addr: *listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		logger.Info("culprit listening", "address", *listen, "repo", *repo)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdown)
}
