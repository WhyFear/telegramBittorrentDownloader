package service

import (
	"telegramBittorrentDownloader/service/cache"
	"telegramBittorrentDownloader/service/downloader"
	"telegramBittorrentDownloader/service/searcher"
)

type Service struct {
	Searcher   map[string]searcher.Searcher
	Downloader map[string]downloader.Downloader
	Cache      *cache.Cache
}
