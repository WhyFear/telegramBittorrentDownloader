package main

import (
	"log/slog"

	"telegramBittorrentDownloader/service"
	"telegramBittorrentDownloader/service/cache"
	downloader2 "telegramBittorrentDownloader/service/downloader"
	searcher2 "telegramBittorrentDownloader/service/searcher"
	"telegramBittorrentDownloader/types"
)

func initSearcher(config *types.Config) map[string]searcher2.Searcher {
	searchers := make(map[string]searcher2.Searcher)
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

func initDownloader(config *types.Config) map[string]downloader2.Downloader {
	downloaders := make(map[string]downloader2.Downloader)
	for _, d := range config.Downloader {
		if !d.Enable {
			continue
		}

		dl, err := downloader2.NewFromConfig(d)
		if err != nil {
			slog.Error("Failed to initialize downloader", "channel", d.Name, "error", err)
			continue
		}
		downloaders[downloader2.NormalizeName(d.Name)] = dl
	}
	return downloaders
}

func initCache() *cache.Cache {
	return cache.NewOtterCache()
}

func InitAll(config *types.Config) *service.Service {
	searchers := initSearcher(config)
	downloaders := initDownloader(config)
	otterCache := initCache()
	return &service.Service{
		Cache:      otterCache,
		Searcher:   searchers,
		Downloader: downloaders,
	}
}
