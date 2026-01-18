package service

import (
	"telegramBittorrentDownloader/service/cache"
	"telegramBittorrentDownloader/service/downloader"
	"telegramBittorrentDownloader/service/searcher"
)

type Service struct {
	Searcher   map[string]*searcher.Search
	Downloader map[string]*downloader.Download
	Cache      *cache.Cache
}
