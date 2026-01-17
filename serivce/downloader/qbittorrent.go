package downloader

import (
	"context"
	"log/slog"
	"telegramBittorrentDownloader/types"

	"github.com/superturkey650/go-qbittorrent/qbt"
)

func NewQBittorrentDownloader(config types.Downloader) *Download {
	qbDownloader := &Download{
		qbittorrent: &QBittorrent{},
	}
	qb := qbt.NewClient(config.ApiURL)
	err := qb.Login(config.Username, config.Password)
	if err != nil {
		slog.Error("Login failed", "err", err, "username", config.Username)
		return nil
	}
	qbDownloader.qbittorrent.QBClient = qb
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

func (d *Download) AddMagnet(ctx context.Context, magnet string) error {
	options := qbt.DownloadOptions{}
	if d.qbittorrent.DownloadOptions != nil {
		options = *d.qbittorrent.DownloadOptions
	}
	magnetLinks := []string{magnet}
	err := d.qbittorrent.QBClient.DownloadLinks(magnetLinks, options)
	if err != nil {
		slog.ErrorContext(ctx, "Add magnet failed", "err", err, "magnet", magnet)
		return err
	}
	slog.InfoContext(ctx, "Add magnet success", "magnet", magnet)
	return nil
}
