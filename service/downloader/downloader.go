package downloader

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Download struct {
	qbittorrent *QBittorrent
}

type QBittorrent struct {
	HTTPClient      *http.Client
	DownloadOptions *DownloadOptions
	ApiURL          string
	Username        string
	Password        string
	lastLoginAt     time.Time
	loginInterval   time.Duration
	mu              sync.Mutex
}

// DownloadOptions contains optional qBittorrent task settings.
type DownloadOptions struct {
	Category *string
	SavePath *string
}

type Downloader interface {
	AddMagnet(ctx context.Context, magnet string) error
}
