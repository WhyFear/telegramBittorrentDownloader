package types

import "net/http"

type Config struct {
	Proxy      Proxy        `yaml:"Proxy"`
	Bot        Bot          `yaml:"bot"`
	Searcher   []Searcher   `yaml:"searcher"`
	Downloader []Downloader `yaml:"downloader"`
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
	Name     string            `yaml:"name"`
	Enable   bool              `yaml:"enable"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	ApiURL   string            `yaml:"api_url"`
	Extra    map[string]string `yaml:"extra"`
}
