package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"telegramBittorrentDownloader/service/downloader"
	"telegramBittorrentDownloader/types"
)

type registeredTestDownloader struct{}

func (d *registeredTestDownloader) AddMagnet(_ context.Context, _ string) error {
	return nil
}

func TestInitDownloaderUsesRegisteredFactory(t *testing.T) {
	const channel = "test-registered-downloader"
	if !downloader.IsRegistered(channel) {
		err := downloader.Register(channel, func(_ types.Downloader) (downloader.Downloader, error) {
			return &registeredTestDownloader{}, nil
		})
		require.NoError(t, err)
	}

	config := &types.Config{
		Downloader: []types.Downloader{{Name: " TEST-REGISTERED-DOWNLOADER ", Enable: true}},
	}

	initialized := initDownloader(config)

	assert.Contains(t, initialized, channel)
}
