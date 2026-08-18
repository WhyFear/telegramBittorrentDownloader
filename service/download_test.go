package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"telegramBittorrentDownloader/service/downloader"
)

const (
	testMagnetHash = "0123456789abcdef0123456789abcdef01234567"
	testMagnetLink = "magnet:?xt=urn:btih:" + testMagnetHash
)

type recordingDownloader struct {
	magnet string
	err    error
}

func (d *recordingDownloader) AddMagnet(_ context.Context, magnet string) error {
	d.magnet = magnet
	return d.err
}

func TestAddMagnetDispatchesByNormalizedChannel(t *testing.T) {
	dl := &recordingDownloader{}
	services := &Service{
		Downloader: map[string]downloader.Downloader{"qbittorrent": dl},
	}
	magnet := testMagnetLink + "&dn=example"

	err := services.AddMagnet(context.Background(), " qBittorrent ", magnet)

	require.NoError(t, err)
	assert.Equal(t, magnet, dl.magnet)
}

func TestAddMagnetExpandsFortyCharacterHash(t *testing.T) {
	dl := &recordingDownloader{}
	services := &Service{
		Downloader: map[string]downloader.Downloader{"qbittorrent": dl},
	}

	err := services.AddMagnet(context.Background(), "qbittorrent", testMagnetHash)

	require.NoError(t, err)
	assert.Equal(t, testMagnetLink, dl.magnet)
}

func TestAddMagnetRejectsInvalidInput(t *testing.T) {
	services := &Service{
		Downloader: map[string]downloader.Downloader{
			"qbittorrent": &recordingDownloader{},
		},
	}
	tests := []struct {
		name    string
		channel string
		magnet  string
		wantErr error
	}{
		{name: "empty channel", channel: "", magnet: testMagnetLink, wantErr: ErrInvalidChannel},
		{name: "empty magnet", channel: "qbittorrent", magnet: "", wantErr: ErrInvalidMagnet},
		{
			name: "non-hex hash", channel: "qbittorrent",
			magnet: strings.Repeat("z", 40), wantErr: ErrInvalidMagnet,
		},
		{
			name: "missing exact topic", channel: "qbittorrent",
			magnet: "magnet:?dn=example", wantErr: ErrInvalidMagnet,
		},
		{
			name: "uppercase scheme", channel: "qbittorrent",
			magnet: "MAGNET:?xt=urn:btih:" + testMagnetHash, wantErr: ErrInvalidMagnet,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := services.AddMagnet(context.Background(), test.channel, test.magnet)
			assert.ErrorIs(t, err, test.wantErr)
		})
	}
}

func TestAddMagnetClassifiesChannelAndDownloaderErrors(t *testing.T) {
	t.Run("unsupported channel", func(t *testing.T) {
		services := &Service{Downloader: make(map[string]downloader.Downloader)}
		err := services.AddMagnet(context.Background(), "missing", testMagnetLink)
		assert.ErrorIs(t, err, ErrUnsupportedChannel)
	})

	t.Run("registered but unavailable channel", func(t *testing.T) {
		services := &Service{Downloader: make(map[string]downloader.Downloader)}
		err := services.AddMagnet(context.Background(), "qbittorrent", testMagnetLink)
		assert.ErrorIs(t, err, ErrChannelUnavailable)
	})

	t.Run("downloader failure", func(t *testing.T) {
		services := &Service{
			Downloader: map[string]downloader.Downloader{
				"qbittorrent": &recordingDownloader{err: errors.New("downstream failure")},
			},
		}
		err := services.AddMagnet(context.Background(), "qbittorrent", testMagnetLink)
		assert.ErrorIs(t, err, ErrDownloadFailed)
	})
}
