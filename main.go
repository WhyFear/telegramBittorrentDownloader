package main

import (
	"context"
	"telegramBittorrentDownloader/bot"
	"telegramBittorrentDownloader/config"
)

func main() {
	sysConfig, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	service := InitAll(sysConfig)
	if service == nil {
		panic("Failed to initialize service")
	}
	bot.InitBot(context.Background(), sysConfig, service)
}
