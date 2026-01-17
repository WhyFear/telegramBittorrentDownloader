package main

import (
	"telegramBittorrentDownloader/serivce"
	downloader2 "telegramBittorrentDownloader/serivce/downloader"
	searcher2 "telegramBittorrentDownloader/serivce/searcher"
	"telegramBittorrentDownloader/types"
)

func initSearcher(config *types.Config) map[string]*searcher2.Search {
	for _, s := range config.Searcher {
		if s.Enable {
			return map[string]*searcher2.Search{
				// todo 按名字初始化
				s.Name: searcher2.NewNyaaSearcher(config.Proxy.Client),
			}
		}
	}
	return nil
}

func initDownloader(config *types.Config) map[string]*downloader2.Download {
	for _, d := range config.Downloader {
		if d.Enable {
			return map[string]*downloader2.Download{
				// todo 按名字初始化
				d.Name: downloader2.NewQBittorrentDownloader(d),
			}
		}
	}
	return nil
}

func InitAll(config *types.Config) *serivce.Service {
	searchers := initSearcher(config)
	downloaders := initDownloader(config)
	return &serivce.Service{
		Searcher:   searchers,
		Downloader: downloaders,
	}
}
