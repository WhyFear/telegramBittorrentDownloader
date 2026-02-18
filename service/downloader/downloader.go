package downloader

import (
	"context"
	"sync"
	"time"

	"github.com/superturkey650/go-qbittorrent/qbt"
)

type Download struct {
	qbittorrent *QBittorrent
}

type QBittorrent struct {
	QBClient        *qbt.Client
	DownloadOptions *qbt.DownloadOptions
	ApiURL          string
	Username        string
	Password        string
	lastLoginAt     time.Time
	loginInterval   time.Duration
	mu              sync.Mutex
}

type Downloader interface {
	AddMagnet(ctx context.Context, magnet string) error
}
