package routes

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"todoapp/internal/config"
)

// Run starts the API server and (when cfg.Server.Monitoring is set) the
// metrics/pprof server in separate goroutines, then blocks until ctx is
// cancelled and shuts everything down gracefully.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger, s *Server) error {
	api := &http.Server{Addr: ":" + cfg.Server.Port, Handler: s.Router()}

	servers := []*http.Server{api}

	if cfg.Server.Monitoring {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		servers = append(servers, &http.Server{Addr: ":" + cfg.Server.MetricsPort, Handler: mux})
	}

	errCh := make(chan error, len(servers))
	for _, srv := range servers {
		go func(srv *http.Server) {
			logger.Info("server listening", "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}(srv)
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range servers {
		_ = srv.Shutdown(shutdownCtx)
	}
	return nil
}
