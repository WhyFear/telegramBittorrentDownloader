package utils

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"telegramBittorrentDownloader/types"
)

func NewProxyClient(config *types.Config) *http.Client {
	if config.Proxy.URL == "" {
		slog.Info("Proxy URL is empty, skip proxy")
		return nil
	}
	proxyURL, err := url.Parse(config.Proxy.URL)
	if err != nil {
		slog.Error("Failed to parse proxy URL", "error", err)
		return nil
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}
	return client
}

// GetUrl 发起请求
func GetUrl(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if client == nil {
		return nil, errors.New("HTTP client is nil")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP request failed with status: " + resp.Status)
	}

	// 检查是否为gzip压缩内容
	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding == "gzip" {
		// 使用gzip reader解压内容
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		content, err := io.ReadAll(gzipReader)
		if err != nil {
			return nil, err
		}
		return content, nil
	}

	// 非压缩内容直接读取
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return content, nil
}
