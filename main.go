package main

import (
	"context"
	"fmt"

	"telegramBittorrentDownloader/api"
	"telegramBittorrentDownloader/bot"
	"telegramBittorrentDownloader/config"
)

func main() {
	sysConfig, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	services := InitAll(sysConfig)
	if services == nil {
		panic("Failed to initialize service")
	}

	apiServer, err := api.NewServer(sysConfig.API, services)
	if err != nil {
		panic(fmt.Errorf("failed to initialize HTTP API: %w", err))
	}
	ctx := context.Background()
	if !apiServer.Enabled() {
		bot.InitBot(ctx, sysConfig, services)
		return
	}

	go bot.InitBot(ctx, sysConfig, services)
	if err := apiServer.ListenAndServe(); err != nil {
		panic(fmt.Errorf("HTTP API stopped: %w", err))
	}
}
