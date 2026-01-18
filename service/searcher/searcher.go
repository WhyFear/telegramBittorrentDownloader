package searcher

import (
	"context"
	"net/http"
	"telegramBittorrentDownloader/types"
)

type Search struct {
	Client *http.Client
}

// Searcher is the interface that wraps the basic methods for searching torrents.
type Searcher interface {
	Search(ctx context.Context, query string) (*types.SearchResult, error)
}
