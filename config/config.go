package config

import (
	"fmt"
	"os"
	"telegramBittorrentDownloader/types"
	"telegramBittorrentDownloader/utils"

	"gopkg.in/yaml.v3"
)

func InitConfig() (*types.Config, error) {
	// 读取配置文件
	config := &types.Config{}
	configPath := "config.yaml"

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 4. 解析YAML
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// 5. 初始化代理客户端
	config.Proxy.Client = utils.NewProxyClient(config)

	return config, nil
}
