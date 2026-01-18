package serivce

import (
	"telegramBittorrentDownloader/serivce/cache"
	"telegramBittorrentDownloader/serivce/downloader"
	"telegramBittorrentDownloader/serivce/searcher"
)

type Service struct {
	Searcher   map[string]*searcher.Search
	Downloader map[string]*downloader.Download
	Cache      *cache.Cache
}
