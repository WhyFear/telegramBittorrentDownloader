package downloader

import (
	"context"
	"errors"
	"log/slog"
	"telegramBittorrentDownloader/types"
	"time"

	"github.com/superturkey650/go-qbittorrent/qbt"
)

func NewQBittorrentDownloader(config types.Downloader) *Download {
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
	err := qbDownloader.qbittorrent.ensureLogin()
	if err != nil {
		slog.Error("Login failed", "err", err, "username", config.Username)
		return nil
	}
	if category, ok := config.Extra["category"]; ok {
		if qbDownloader.qbittorrent.DownloadOptions == nil {
			qbDownloader.qbittorrent.DownloadOptions = &qbt.DownloadOptions{}
		}
		qbDownloader.qbittorrent.DownloadOptions.Category = &category
	}
	if savePath, ok := config.Extra["save_path"]; ok {
		if qbDownloader.qbittorrent.DownloadOptions == nil {
			qbDownloader.qbittorrent.DownloadOptions = &qbt.DownloadOptions{}
		}
		qbDownloader.qbittorrent.DownloadOptions.Savepath = &savePath
	}
	return qbDownloader
}

func (q *QBittorrent) ensureLogin() error {
	if q == nil {
		return errors.New("qbittorrent client is nil")
	}
	if q.ApiURL == "" {
		return errors.New("qbittorrent api url is empty")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.QBClient == nil {
		q.QBClient = qbt.NewClient(q.ApiURL)
	}
	if q.loginInterval <= 0 {
		q.loginInterval = 30 * time.Minute
	}
	if !q.lastLoginAt.IsZero() && time.Since(q.lastLoginAt) < q.loginInterval {
		return nil
	}
	if err := q.QBClient.Login(q.Username, q.Password); err != nil {
		return err
	}
	q.lastLoginAt = time.Now()
	return nil
}

func (d *Download) AddMagnet(ctx context.Context, magnet string) error {
	if d == nil || d.qbittorrent == nil {
		err := errors.New("qbittorrent downloader not initialized")
		slog.ErrorContext(ctx, "Add magnet failed", "err", err, "magnet", magnet)
		return err
	}
	err := d.qbittorrent.ensureLogin()
	if err != nil {
		slog.ErrorContext(ctx, "Login failed", "err", err)
		return err
	}
	options := qbt.DownloadOptions{}
	if d.qbittorrent.DownloadOptions != nil {
		options = *d.qbittorrent.DownloadOptions
	}
	magnetLinks := []string{magnet}
	err = d.qbittorrent.QBClient.DownloadLinks(magnetLinks, options)
	if err != nil {
		slog.ErrorContext(ctx, "Add magnet failed", "err", err, "magnet", magnet)
		return err
	}
	slog.InfoContext(ctx, "Add magnet success", "magnet", magnet)
	return nil
}
