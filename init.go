package main

import (
	"telegramBittorrentDownloader/serivce"
	"telegramBittorrentDownloader/serivce/cache"
	downloader2 "telegramBittorrentDownloader/serivce/downloader"
	searcher2 "telegramBittorrentDownloader/serivce/searcher"
	"telegramBittorrentDownloader/types"
)

func initSearcher(config *types.Config) map[string]*searcher2.Search {
	searchers := make(map[string]*searcher2.Search)
	for _, s := range config.Searcher {
		if s.Enable {
			if s.Name == "nyaa" {
				searchers[s.Name] = searcher2.NewNyaaSearcher(config.Proxy.Client)
			}
			// todo 可以在这里添加其他搜索器的初始化逻辑
		}
	}
	return searchers
}

func initDownloader(config *types.Config) map[string]*downloader2.Download {
	downloaders := make(map[string]*downloader2.Download)
	for _, d := range config.Downloader {
		if d.Enable {
			if d.Name == "qbittorrent" {
				downloaders[d.Name] = downloader2.NewQBittorrentDownloader(d)
			}
			// todo 可以在这里添加其他下载器的初始化逻辑
		}
	}
	return downloaders
}

func initCache() *cache.Cache {
	return cache.NewOtterCache()
}

func InitAll(config *types.Config) *serivce.Service {
	searchers := initSearcher(config)
	downloaders := initDownloader(config)
	otterCache := initCache()
	return &serivce.Service{
		Cache:      otterCache,
		Searcher:   searchers,
		Downloader: downloaders,
	}
}
