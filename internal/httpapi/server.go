package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"chat-trace-viewer/internal/config"
	"chat-trace-viewer/internal/domain"
	"chat-trace-viewer/internal/service"
)

type Server struct {
	cfg     config.Config
	logger  *slog.Logger
	service *service.ChatTraceService
}

func New(cfg config.Config, logger *slog.Logger, service *service.ChatTraceService) *Server {
	return &Server{cfg: cfg, logger: logger, service: service}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/chat-trace", s.handleChatTrace)

	webDir := filepath.Join(".", "web")
	if _, err := os.Stat(webDir); err == nil {
		fs := http.FileServer(http.Dir(webDir))
		mux.Handle("/", fs)
	}

	return s.logRequests(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	configured := strings.TrimSpace(s.cfg.VictoriaLogsBaseURL) != ""

	writeJSON(w, http.StatusOK, map[string]any{
		"default_lookback":        s.cfg.DefaultLookback.String(),
		"max_log_lines":           s.cfg.MaxLogLines,
		"victorialogs_configured": configured,
		"victorialogs_base_url":   s.cfg.VictoriaLogsBaseURL,
		"victorialogs_account_id": s.cfg.VictoriaLogsAccountID,
		"victorialogs_project_id": s.cfg.VictoriaLogsProjectID,
	})
}

func (s *Server) handleChatTrace(w http.ResponseWriter, r *http.Request) {
	query, err := readTraceQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.VictoriaLogsTimeout)
	defer cancel()

	trace, err := s.service.BuildChatTrace(ctx, query)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

func readTraceQuery(r *http.Request) (domain.TraceQuery, error) {
	values := r.URL.Query()
	query := domain.TraceQuery{
		UserID:             strings.TrimSpace(values.Get("user_id")),
		SessionID:          strings.TrimSpace(values.Get("session_id")),
		Project:            strings.TrimSpace(values.Get("project")),
		Client:             strings.TrimSpace(values.Get("client")),
		HideDebug:          parseBool(values.Get("hide_debug")),
		OnlyAnomalies:      parseBool(values.Get("only_anomalies")),
		CollapseDuplicates: parseBool(values.Get("collapse_duplicates")),
	}

	if from := strings.TrimSpace(values.Get("from")); from != "" {
		ts, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return query, err
		}
		query.From = &ts
	}
	if to := strings.TrimSpace(values.Get("to")); to != "" {
		ts, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return query, err
		}
		query.To = &ts
	}
	return query, nil
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func optionalInt(values map[string][]string, key string) int {
	v := strings.TrimSpace(first(values[key]))
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
