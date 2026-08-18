package types

import "net/http"

type Config struct {
	// SenderID 是允许使用该 bot 的用户 ID 列表
	// 如果 SenderID 为空，则表示允许所有用户使用该 bot
	SenderID   []int64      `yaml:"SenderID"`
	Proxy      Proxy        `yaml:"Proxy"`
	Bot        Bot          `yaml:"bot"`
	API        APIConfig    `yaml:"api"`
	Searcher   []Searcher   `yaml:"searcher"`
	Downloader []Downloader `yaml:"downloader"`
}

// APIConfig controls the optional HTTP download API.
type APIConfig struct {
	ListenIP string `yaml:"listen_ip"`
	Port     int    `yaml:"port"`
	Token    string `yaml:"token"`
}

type Proxy struct {
	URL    string       `yaml:"URL"`
	Client *http.Client `yaml:"-"`
}

type Bot struct {
	BotName string `yaml:"bot_name"`
	Token   string `yaml:"token"`
}

type Searcher struct {
	Name   string `yaml:"name"`
	Enable bool   `yaml:"enable"`
}

type Downloader struct {
	Name                 string            `yaml:"name"`
	Enable               bool              `yaml:"enable"`
	Username             string            `yaml:"username"`
	Password             string            `yaml:"password"`
	ApiURL               string            `yaml:"api_url"`
	LoginIntervalSeconds int               `yaml:"login_interval_seconds"`
	Extra                map[string]string `yaml:"extra"`
}
