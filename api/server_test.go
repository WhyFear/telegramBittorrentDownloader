package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"telegramBittorrentDownloader/service"
	"telegramBittorrentDownloader/types"
)

const validMagnet = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"

type recordingMagnetAdder struct {
	called  bool
	channel string
	magnet  string
	err     error
}

func (a *recordingMagnetAdder) AddMagnet(_ context.Context, channel string, magnet string) error {
	a.called = true
	a.channel = channel
	a.magnet = magnet
	return a.err
}

func TestNewServerConfig(t *testing.T) {
	t.Run("port zero disables API", func(t *testing.T) {
		server, err := NewServer(types.APIConfig{Port: 0}, &recordingMagnetAdder{})
		require.NoError(t, err)
		assert.False(t, server.Enabled())
		assert.ErrorIs(t, server.ListenAndServe(), ErrServerDisabled)
	})

	t.Run("empty IP defaults to loopback", func(t *testing.T) {
		server, err := NewServer(types.APIConfig{Port: 8081}, &recordingMagnetAdder{})
		require.NoError(t, err)
		assert.True(t, server.Enabled())
		assert.Equal(t, "127.0.0.1:8081", server.address)
	})

	tests := []struct {
		name   string
		config types.APIConfig
	}{
		{name: "negative port", config: types.APIConfig{Port: -1}},
		{name: "port too large", config: types.APIConfig{Port: 65536}},
		{name: "invalid enabled IP", config: types.APIConfig{ListenIP: "localhost", Port: 8081}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewServer(test.config, &recordingMagnetAdder{})
			assert.Error(t, err)
		})
	}
}

func TestDownloadEndpointAcceptsRequestWithoutConfiguredToken(t *testing.T) {
	adder := &recordingMagnetAdder{}
	server := mustTestServer(t, "", adder)

	response := performRequest(server, http.MethodPost, validPayload(" qBittorrent "), "")

	assert.Equal(t, http.StatusAccepted, response.Code)
	assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	assert.True(t, adder.called)
	assert.Equal(t, "qbittorrent", adder.channel)
	assert.Equal(t, validMagnet, adder.magnet)
	assert.Equal(
		t,
		map[string]string{"status": "accepted", "channel": "qbittorrent"},
		decodeBody(t, response),
	)
}

func TestDownloadEndpointBearerAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", header: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "malformed header", header: "Token secret", wantStatus: http.StatusUnauthorized},
		{
			name: "correct token", header: "Bearer secret",
			wantStatus: http.StatusAccepted, wantCalled: true,
		},
		{
			name: "case-insensitive scheme", header: "bearer secret",
			wantStatus: http.StatusAccepted, wantCalled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adder := &recordingMagnetAdder{}
			server := mustTestServer(t, "secret", adder)
			response := performRequest(server, http.MethodPost, validPayload("qbittorrent"), test.header)

			assert.Equal(t, test.wantStatus, response.Code)
			assert.Equal(t, test.wantCalled, adder.called)
			if test.wantStatus == http.StatusUnauthorized {
				assert.Equal(t, "Bearer", response.Header().Get("WWW-Authenticate"))
				assert.Equal(t, "unauthorized", decodeBody(t, response)["code"])
			}
		})
	}
}

func TestDownloadEndpointRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: "{"},
		{
			name: "unknown field",
			body: `{"magnet":"` + validMagnet + `","channel":"qbittorrent","extra":true}`,
		},
		{name: "multiple objects", body: validPayload("qbittorrent") + `{}`},
		{name: "missing magnet", body: `{"channel":"qbittorrent"}`},
		{name: "missing channel", body: `{"magnet":"` + validMagnet + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adder := &recordingMagnetAdder{}
			server := mustTestServer(t, "", adder)
			response := performRequest(server, http.MethodPost, test.body, "")

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.False(t, adder.called)
			assert.Equal(t, "invalid_request", decodeBody(t, response)["code"])
		})
	}
}

func TestDownloadEndpointRejectsOversizedRequest(t *testing.T) {
	adder := &recordingMagnetAdder{}
	server := mustTestServer(t, "", adder)
	body := `{"magnet":"` + strings.Repeat("a", maxRequestBytes) +
		`","channel":"qbittorrent"}`

	response := performRequest(server, http.MethodPost, body, "")

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.False(t, adder.called)
	assert.Equal(t, "invalid_request", decodeBody(t, response)["code"])
}

func TestDownloadEndpointMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name: "invalid Magnet", err: service.ErrInvalidMagnet,
			wantStatus: http.StatusBadRequest, wantCode: "invalid_request",
		},
		{
			name: "unsupported channel", err: service.ErrUnsupportedChannel,
			wantStatus: http.StatusNotFound, wantCode: "unsupported_channel",
		},
		{
			name: "unavailable channel", err: service.ErrChannelUnavailable,
			wantStatus: http.StatusServiceUnavailable, wantCode: "channel_unavailable",
		},
		{
			name: "downloader failure", err: service.ErrDownloadFailed,
			wantStatus: http.StatusBadGateway, wantCode: "download_failed",
		},
		{
			name: "unexpected failure", err: errors.New("internal details"),
			wantStatus: http.StatusBadGateway, wantCode: "download_failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := mustTestServer(t, "", &recordingMagnetAdder{err: test.err})
			response := performRequest(server, http.MethodPost, validPayload("qbittorrent"), "")

			assert.Equal(t, test.wantStatus, response.Code)
			body := decodeBody(t, response)
			assert.Equal(t, test.wantCode, body["code"])
			assert.NotContains(t, body["message"], "internal details")
		})
	}
}

func TestDownloadEndpointRejectsUnsupportedMethodBeforeAuthentication(t *testing.T) {
	server := mustTestServer(t, "secret", &recordingMagnetAdder{})

	response := performRequest(server, http.MethodGet, "", "")

	assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
	assert.Equal(t, http.MethodPost, response.Header().Get("Allow"))
	assert.Equal(t, "method_not_allowed", decodeBody(t, response)["code"])
}

func mustTestServer(t *testing.T, token string, adder MagnetAdder) *Server {
	t.Helper()
	server, err := NewServer(types.APIConfig{Token: token}, adder)
	require.NoError(t, err)
	return server
}

func performRequest(
	server *Server,
	method string,
	body string,
	authorization string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/v1/downloads", strings.NewReader(body))
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func validPayload(channel string) string {
	return `{"magnet":"` + validMagnet + `","channel":"` + channel + `"}`
}

func decodeBody(t *testing.T, response *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	return body
}
