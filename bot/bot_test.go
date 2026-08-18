package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"telegramBittorrentDownloader/service"
	"telegramBittorrentDownloader/service/downloader"
)

const (
	testMagnetHash = "0123456789abcdef0123456789abcdef01234567"
	testMagnetLink = "magnet:?xt=urn:btih:" + testMagnetHash
)

type recordingDownloader struct {
	magnet string
}

func (d *recordingDownloader) AddMagnet(_ context.Context, magnet string) error {
	d.magnet = magnet
	return nil
}

func TestIsStartDownloadPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{
			name:    "40 character hash",
			payload: testMagnetHash,
			want:    true,
		},
		{
			name:    "magnet link",
			payload: testMagnetLink,
			want:    true,
		},
		{
			name:    "magnet link with parameters",
			payload: testMagnetLink + "&dn=example&tr=https%3A%2F%2Ftracker.example",
			want:    true,
		},
		{
			name:    "search query",
			payload: "example torrent",
			want:    false,
		},
		{
			name:    "empty payload",
			payload: "",
			want:    false,
		},
		{
			name:    "URL encoded magnet link",
			payload: "magnet%3A%3Fxt%3Durn%3Abtih%3A" + testMagnetHash,
			want:    false,
		},
		{
			name:    "uppercase magnet scheme",
			payload: "MAGNET:?xt=urn:btih:" + testMagnetHash,
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isStartDownloadPayload(test.payload))
		})
	}
}

func TestAddMagnetPassesDirectLinkToDownloader(t *testing.T) {
	dl := &recordingDownloader{}
	services := &service.Service{
		Downloader: map[string]downloader.Downloader{
			"qbittorrent": dl,
		},
	}
	magnet := testMagnetLink + "&dn=example"

	err := addMagnet(context.Background(), magnet, services)

	assert.NoError(t, err)
	assert.Equal(t, magnet, dl.magnet)
}
