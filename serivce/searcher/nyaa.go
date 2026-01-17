package searcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"telegramBittorrentDownloader/types"
	"telegramBittorrentDownloader/utils"
)

func NewNyaaSearcher(client *http.Client) *Search {
	return &Search{
		Client: client,
	}
}

type NyaaResponse struct {
	Count int              `json:"count"`
	Data  []*types.Torrent `json:"data"`
}

// https://github.com/Vivek-Kolhe/Nyaa-API
const NyaaSearchURL = "https://nyaaapi.onrender.com/sukebei?q=%s&sort=seeders"

func (s *Search) Search(ctx context.Context, query string) (*types.SearchResult, error) {
	searchUrl := fmt.Sprintf(NyaaSearchURL, query)
	data, err := utils.GetUrl(ctx, s.Client, searchUrl)
	if err != nil {
		return nil, err
	}

	var nyaaResp NyaaResponse
	if err = json.Unmarshal(data, &nyaaResp); err != nil {
		return nil, err
	}

	result := &types.SearchResult{
		Count: nyaaResp.Count,
		Data:  nyaaResp.Data,
	}

	return result, nil
}
