// Package api exposes the download service through an authenticated HTTP API.
package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"telegramBittorrentDownloader/service"
	"telegramBittorrentDownloader/types"
)

const (
	defaultListenIP = "127.0.0.1"
	maxRequestBytes = 64 << 10
)

// ErrServerDisabled indicates that ListenAndServe was called with port zero.
var ErrServerDisabled = errors.New("HTTP API is disabled")

// MagnetAdder accepts a Magnet task for a named downloader channel.
type MagnetAdder interface {
	AddMagnet(ctx context.Context, channel string, magnet string) error
}

// Server owns the HTTP handler and listener configuration for the download API.
type Server struct {
	address string
	enabled bool
	token   string
	service MagnetAdder
	handler http.Handler
	server  *http.Server
}

// NewServer validates config and creates an HTTP API server.
func NewServer(config types.APIConfig, downloadService MagnetAdder) (*Server, error) {
	if config.Port < 0 || config.Port > 65535 {
		return nil, fmt.Errorf("API port must be between 0 and 65535")
	}
	if downloadService == nil {
		return nil, errors.New("download service cannot be nil")
	}

	listenIP := strings.TrimSpace(config.ListenIP)
	if listenIP == "" {
		listenIP = defaultListenIP
	}
	if config.Port > 0 && net.ParseIP(listenIP) == nil {
		return nil, fmt.Errorf("invalid API listen IP %q", listenIP)
	}

	apiServer := &Server{
		address: net.JoinHostPort(listenIP, strconv.Itoa(config.Port)),
		enabled: config.Port > 0,
		token:   config.Token,
		service: downloadService,
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/downloads", apiServer.methodOnly(
		http.MethodPost,
		apiServer.authenticate(http.HandlerFunc(apiServer.handleDownload)),
	))
	apiServer.handler = mux
	apiServer.server = &http.Server{
		Addr:              apiServer.address,
		Handler:           apiServer.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return apiServer, nil
}

// Enabled reports whether the HTTP API has a non-zero listening port.
func (s *Server) Enabled() bool {
	return s != nil && s.enabled
}

// Handler returns the API's HTTP handler for embedding and tests.
func (s *Server) Handler() http.Handler {
	return s.handler
}

// ListenAndServe starts the configured HTTP API listener.
func (s *Server) ListenAndServe() error {
	if !s.Enabled() {
		return ErrServerDisabled
	}
	slog.Info("HTTP API started", "address", s.address, "authentication", s.token != "")
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) methodOnly(method string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != method {
			response.Header().Set("Allow", method)
			writeError(
				response,
				request,
				http.StatusMethodNotAllowed,
				"method_not_allowed",
				"method not allowed",
			)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if s.token == "" {
			next.ServeHTTP(response, request)
			return
		}

		providedToken, ok := parseBearerToken(request.Header.Get("Authorization"))
		expectedHash := sha256.Sum256([]byte(s.token))
		providedHash := sha256.Sum256([]byte(providedToken))
		if !ok || subtle.ConstantTimeCompare(expectedHash[:], providedHash[:]) != 1 {
			response.Header().Set("WWW-Authenticate", "Bearer")
			writeError(
				response,
				request,
				http.StatusUnauthorized,
				"unauthorized",
				"missing or invalid bearer token",
			)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (s *Server) handleDownload(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var payload downloadRequest
	if err := decoder.Decode(&payload); err != nil {
		writeError(
			response,
			request,
			http.StatusBadRequest,
			"invalid_request",
			"request body must be a JSON object",
		)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(
			response,
			request,
			http.StatusBadRequest,
			"invalid_request",
			"request body must contain one JSON object",
		)
		return
	}
	if strings.TrimSpace(payload.Channel) == "" || strings.TrimSpace(payload.Magnet) == "" {
		writeError(
			response,
			request,
			http.StatusBadRequest,
			"invalid_request",
			"magnet and channel are required",
		)
		return
	}

	channel := service.NormalizeChannel(payload.Channel)
	err := s.service.AddMagnet(request.Context(), channel, payload.Magnet)
	if err != nil {
		slog.ErrorContext(
			request.Context(),
			"HTTP download request failed",
			"channel",
			channel,
			"error",
			err,
		)
		switch {
		case errors.Is(err, service.ErrInvalidChannel), errors.Is(err, service.ErrInvalidMagnet):
			writeError(
				response,
				request,
				http.StatusBadRequest,
				"invalid_request",
				"magnet or channel is invalid",
			)
		case errors.Is(err, service.ErrUnsupportedChannel):
			writeError(
				response,
				request,
				http.StatusNotFound,
				"unsupported_channel",
				"requested channel is not supported",
			)
		case errors.Is(err, service.ErrChannelUnavailable):
			writeError(
				response,
				request,
				http.StatusServiceUnavailable,
				"channel_unavailable",
				"requested channel is unavailable",
			)
		default:
			writeError(
				response,
				request,
				http.StatusBadGateway,
				"download_failed",
				"downloader failed to accept the task",
			)
		}
		return
	}

	writeJSON(response, request, http.StatusAccepted, downloadResponse{
		Status:  "accepted",
		Channel: channel,
	})
}

func parseBearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func writeError(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	code string,
	message string,
) {
	writeJSON(response, request, status, errorResponse{Code: code, Message: message})
}

func writeJSON(response http.ResponseWriter, request *http.Request, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	if err := json.NewEncoder(response).Encode(payload); err != nil {
		slog.ErrorContext(request.Context(), "Failed to write HTTP API response", "error", err)
	}
}

type downloadRequest struct {
	Magnet  string `json:"magnet"`
	Channel string `json:"channel"`
}

type downloadResponse struct {
	Status  string `json:"status"`
	Channel string `json:"channel"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
