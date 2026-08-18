package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"telegramBittorrentDownloader/types"
)

const maxQBittorrentResponseBytes = 64 << 10

func init() {
	err := Register("qbittorrent", func(config types.Downloader) (Downloader, error) {
		return NewQBittorrentDownloader(config)
	})
	if err != nil {
		slog.Error("Failed to register qBittorrent downloader", "error", err)
	}
}

// NewQBittorrentDownloader initializes a qBittorrent-backed downloader.
func NewQBittorrentDownloader(config types.Downloader) (*Download, error) {
	loginInterval := 30 * time.Minute
	if config.LoginIntervalSeconds > 0 {
		loginInterval = time.Duration(config.LoginIntervalSeconds) * time.Second
	}
	qbDownloader := &Download{
		qbittorrent: &QBittorrent{
			ApiURL:        config.ApiURL,
			Username:      config.Username,
			Password:      config.Password,
			loginInterval: loginInterval,
		},
	}
	if err := qbDownloader.qbittorrent.ensureLogin(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to log in to qBittorrent: %w", err)
	}
	if category, ok := config.Extra["category"]; ok {
		if qbDownloader.qbittorrent.DownloadOptions == nil {
			qbDownloader.qbittorrent.DownloadOptions = &DownloadOptions{}
		}
		qbDownloader.qbittorrent.DownloadOptions.Category = &category
	}
	if savePath, ok := config.Extra["save_path"]; ok {
		if qbDownloader.qbittorrent.DownloadOptions == nil {
			qbDownloader.qbittorrent.DownloadOptions = &DownloadOptions{}
		}
		qbDownloader.qbittorrent.DownloadOptions.SavePath = &savePath
	}
	return qbDownloader, nil
}

func (q *QBittorrent) ensureLogin(ctx context.Context) error {
	if q == nil {
		return errors.New("qbittorrent client is nil")
	}
	if strings.TrimSpace(q.ApiURL) == "" {
		return errors.New("qbittorrent api url is empty")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if err := q.ensureHTTPClient(); err != nil {
		return err
	}
	if q.loginInterval <= 0 {
		q.loginInterval = 30 * time.Minute
	}
	if !q.lastLoginAt.IsZero() && time.Since(q.lastLoginAt) < q.loginInterval {
		return nil
	}
	form := url.Values{}
	form.Set("username", q.Username)
	form.Set("password", q.Password)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		q.endpoint("/api/v2/auth/login"),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create qBittorrent login request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "telegramBittorrentDownloader")

	statusCode, responseBody, err := q.do(request)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("qBittorrent login returned HTTP %d", statusCode)
	}
	if strings.TrimSpace(string(responseBody)) != "Ok." {
		return errors.New("qBittorrent rejected the login credentials")
	}
	q.lastLoginAt = time.Now()
	return nil
}

func (d *Download) AddMagnet(ctx context.Context, magnet string) error {
	if d == nil || d.qbittorrent == nil {
		err := errors.New("qbittorrent downloader not initialized")
		slog.ErrorContext(ctx, "Add magnet failed", "error", err)
		return err
	}
	err := d.qbittorrent.ensureLogin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Login failed", "err", err)
		return err
	}
	if err := d.qbittorrent.addMagnet(ctx, magnet); err != nil {
		slog.ErrorContext(ctx, "Add magnet failed", "error", err)
		return err
	}
	slog.InfoContext(ctx, "Add magnet success")
	return nil
}

func (q *QBittorrent) ensureHTTPClient() error {
	if q.HTTPClient == nil {
		q.HTTPClient = &http.Client{}
	}
	if q.HTTPClient.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return fmt.Errorf("failed to create qBittorrent cookie jar: %w", err)
		}
		q.HTTPClient.Jar = jar
	}
	if q.HTTPClient.CheckRedirect == nil {
		q.HTTPClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	return nil
}

func (q *QBittorrent) addMagnet(ctx context.Context, magnet string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("urls", magnet); err != nil {
		return fmt.Errorf("failed to write Magnet field: %w", err)
	}
	if q.DownloadOptions != nil {
		if q.DownloadOptions.Category != nil {
			if err := writer.WriteField("category", *q.DownloadOptions.Category); err != nil {
				return fmt.Errorf("failed to write category field: %w", err)
			}
		}
		if q.DownloadOptions.SavePath != nil {
			if err := writer.WriteField("savepath", *q.DownloadOptions.SavePath); err != nil {
				return fmt.Errorf("failed to write save path field: %w", err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finish qBittorrent request body: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		q.endpoint("/api/v2/torrents/add"),
		&body,
	)
	if err != nil {
		return fmt.Errorf("failed to create qBittorrent add request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Referer", q.ApiURL)
	request.Header.Set("User-Agent", "telegramBittorrentDownloader")

	statusCode, _, err := q.do(request)
	if err != nil {
		return err
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"qBittorrent add request returned HTTP %d %s",
			statusCode,
			http.StatusText(statusCode),
		)
	}
	return nil
}

func (q *QBittorrent) do(request *http.Request) (int, []byte, error) {
	response, err := q.HTTPClient.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("qBittorrent request failed: %w", err)
	}

	responseBody, readErr := io.ReadAll(io.LimitReader(
		response.Body,
		maxQBittorrentResponseBytes+1,
	))
	closeErr := response.Body.Close()
	if readErr != nil {
		return 0, nil, fmt.Errorf("failed to read qBittorrent response: %w", readErr)
	}
	if closeErr != nil {
		return 0, nil, fmt.Errorf("failed to close qBittorrent response: %w", closeErr)
	}
	if len(responseBody) > maxQBittorrentResponseBytes {
		return 0, nil, errors.New("qBittorrent response body is too large")
	}
	return response.StatusCode, responseBody, nil
}

func (q *QBittorrent) endpoint(path string) string {
	return strings.TrimRight(strings.TrimSpace(q.ApiURL), "/") + path
}
