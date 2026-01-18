package downloader

import (
	"context"

	"github.com/superturkey650/go-qbittorrent/qbt"
)

type Download struct {
	qbittorrent *QBittorrent
}

type QBittorrent struct {
	QBClient        *qbt.Client
	DownloadOptions *qbt.DownloadOptions
}

type Downloader interface {
	AddMagnet(ctx context.Context, magnet string) error
}
