package downloader

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"telegramBittorrentDownloader/types"
)

const testQBittorrentMagnet = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"

func TestNewQBittorrentDownloaderValidatesLoginResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "server error", statusCode: http.StatusInternalServerError, body: "error"},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, body: "unauthorized"},
		{name: "rejected credentials", statusCode: http.StatusOK, body: "Fails."},
		{name: "redirect", statusCode: http.StatusFound, body: "redirect"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(
				response http.ResponseWriter,
				request *http.Request,
			) {
				assert.Equal(t, "/api/v2/auth/login", request.URL.Path)
				if test.statusCode == http.StatusFound {
					response.Header().Set("Location", "/login")
				}
				response.WriteHeader(test.statusCode)
				writeTestResponse(t, response, test.body)
			}))
			defer server.Close()

			_, err := NewQBittorrentDownloader(types.Downloader{
				ApiURL: server.URL,
			})

			assert.Error(t, err)
		})
	}
}

func TestQBittorrentAddMagnetSendsConfiguredFields(t *testing.T) {
	var (
		loginUsername string
		loginPassword string
		addCookie     string
		addMagnet     string
		addCategory   string
		addSavePath   string
	)
	server := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		switch request.URL.Path {
		case "/api/v2/auth/login":
			loginUsername = request.FormValue("username")
			loginPassword = request.FormValue("password")
			http.SetCookie(response, &http.Cookie{
				Name: "SID", Value: "session", Path: "/",
			})
			writeTestResponse(t, response, "Ok.")
		case "/api/v2/torrents/add":
			if cookie, err := request.Cookie("SID"); err == nil {
				addCookie = cookie.Value
			}
			addMagnet = request.FormValue("urls")
			addCategory = request.FormValue("category")
			addSavePath = request.FormValue("savepath")
			writeTestResponse(t, response, "Ok.")
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	dl, err := NewQBittorrentDownloader(types.Downloader{
		ApiURL:   server.URL + "/",
		Username: "admin",
		Password: "password",
		Extra: map[string]string{
			"category":  "anime",
			"save_path": "/downloads",
		},
	})
	require.NoError(t, err)

	err = dl.AddMagnet(context.Background(), testQBittorrentMagnet)

	require.NoError(t, err)
	assert.Equal(t, "admin", loginUsername)
	assert.Equal(t, "password", loginPassword)
	assert.Equal(t, "session", addCookie)
	assert.Equal(t, testQBittorrentMagnet, addMagnet)
	assert.Equal(t, "anime", addCategory)
	assert.Equal(t, "/downloads", addSavePath)
}

func TestQBittorrentAddMagnetRejectsHTTPErrorStatus(t *testing.T) {
	tests := []int{
		http.StatusForbidden,
		http.StatusUnsupportedMediaType,
		http.StatusInternalServerError,
	}
	for _, statusCode := range tests {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(
				response http.ResponseWriter,
				request *http.Request,
			) {
				if request.URL.Path == "/api/v2/auth/login" {
					writeTestResponse(t, response, "Ok.")
					return
				}
				response.WriteHeader(statusCode)
				writeTestResponse(t, response, "downstream details")
			}))
			defer server.Close()

			dl, err := NewQBittorrentDownloader(types.Downloader{ApiURL: server.URL})
			require.NoError(t, err)

			err = dl.AddMagnet(context.Background(), testQBittorrentMagnet)

			assert.ErrorContains(t, err, http.StatusText(statusCode))
			assert.NotContains(t, err.Error(), "downstream details")
		})
	}
}

func TestQBittorrentClosesAddResponseBody(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK},
		{name: "failure", statusCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackingResponseBody{Reader: strings.NewReader("response")}
			q := &QBittorrent{
				ApiURL: "http://qbittorrent.example",
				HTTPClient: &http.Client{Transport: roundTripFunc(func(
					_ *http.Request,
				) (*http.Response, error) {
					return &http.Response{
						StatusCode: test.statusCode,
						Header:     make(http.Header),
						Body:       body,
					}, nil
				})},
				lastLoginAt:   time.Now(),
				loginInterval: time.Hour,
			}
			dl := &Download{qbittorrent: q}

			err := dl.AddMagnet(context.Background(), testQBittorrentMagnet)

			assert.Equal(t, test.wantErr, err != nil)
			assert.True(t, body.closed)
		})
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackingResponseBody struct {
	io.Reader
	closed bool
}

func (b *trackingResponseBody) Close() error {
	b.closed = true
	return nil
}

func writeTestResponse(t *testing.T, response http.ResponseWriter, body string) {
	_, err := io.WriteString(response, body)
	assert.NoError(t, err)
}
