package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type tunnelService interface {
	StartTunnel(context.Context, StartTunnelRequest) (TunnelStatus, error)
	ListTunnels() []TunnelStatus
	GetTunnel(string) (TunnelStatus, error)
	StopTunnel(context.Context, string) (TunnelStatus, error)
	Events(string) []TunnelEvent
}

type Server struct {
	service tunnelService
	token   string
}

func NewServer(service tunnelService, token string) *Server {
	return &Server{
		service: service,
		token:   strings.TrimSpace(token),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/tunnels", s.handleTunnels)
	mux.HandleFunc("/api/v1/tunnels/", s.handleTunnel)
	mux.HandleFunc("/api/v1/events", s.handleEvents)

	return s.withAuth(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"tunnels": s.service.ListTunnels()})
	case http.MethodPost:
		var request StartTunnelRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		status, err := s.service.StartTunnel(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, status)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	id, action, ok := parseTunnelPath(strings.TrimPrefix(r.URL.Path, "/api/v1/tunnels/"))
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if action == "shutdown" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		s.stopTunnel(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		status, err := s.service.GetTunnel(id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case http.MethodDelete:
		s.stopTunnel(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodDelete)
	}
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status, err := s.service.StopTunnel(ctx, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": s.service.Events(r.URL.Query().Get("tunnel_id")),
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	if s.token == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.token {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseTunnelPath(path string) (id string, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] != "" {
		return parts[0], "", true
	}
	if len(parts) == 2 && parts[0] != "" && parts[1] == "shutdown" {
		return parts[0], parts[1], true
	}
	return "", "", false
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrTunnelNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, fmt.Sprintf("method must be one of: %s", strings.Join(methods, ", ")))
}
