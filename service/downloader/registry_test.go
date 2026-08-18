package downloader

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"telegramBittorrentDownloader/types"
)

func TestQBittorrentFactoryIsRegistered(t *testing.T) {
	assert.True(t, IsRegistered(" qBittorrent "))
}

func TestNewFromConfigRejectsUnknownDownloader(t *testing.T) {
	_, err := NewFromConfig(types.Downloader{Name: "missing-downloader"})
	assert.ErrorIs(t, err, ErrDownloaderNotRegistered)
}
