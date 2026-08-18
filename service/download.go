package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"telegramBittorrentDownloader/service/downloader"
)

var (
	// ErrInvalidChannel indicates that the requested downloader channel is empty.
	ErrInvalidChannel = errors.New("downloader channel cannot be empty")
	// ErrInvalidMagnet indicates that the supplied Magnet link or hash is invalid.
	ErrInvalidMagnet = errors.New("invalid Magnet link or hash")
	// ErrUnsupportedChannel indicates that no downloader implementation is registered.
	ErrUnsupportedChannel = errors.New("downloader channel is not supported")
	// ErrChannelUnavailable indicates that a registered downloader is not initialized.
	ErrChannelUnavailable = errors.New("downloader channel is unavailable")
	// ErrDownloadFailed indicates that the selected downloader rejected the task.
	ErrDownloadFailed = errors.New("downloader failed to add Magnet")
)

// NormalizeChannel returns the canonical downloader channel name.
func NormalizeChannel(channel string) string {
	return downloader.NormalizeName(channel)
}

// AddMagnet validates and dispatches a Magnet task to an initialized downloader.
func (s *Service) AddMagnet(ctx context.Context, channel string, magnet string) error {
	channel = NormalizeChannel(channel)
	if channel == "" {
		return ErrInvalidChannel
	}

	normalizedMagnet, err := normalizeMagnet(magnet)
	if err != nil {
		return err
	}
	if !downloader.IsRegistered(channel) {
		return fmt.Errorf("%w: %s", ErrUnsupportedChannel, channel)
	}
	if s == nil {
		return fmt.Errorf("%w: %s", ErrChannelUnavailable, channel)
	}

	dl, exists := s.Downloader[channel]
	if !exists || dl == nil {
		return fmt.Errorf("%w: %s", ErrChannelUnavailable, channel)
	}
	if err := dl.AddMagnet(ctx, normalizedMagnet); err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	return nil
}

func normalizeMagnet(magnet string) (string, error) {
	magnet = strings.TrimSpace(magnet)
	if magnet == "" {
		return "", ErrInvalidMagnet
	}

	if len(magnet) == 40 {
		if _, err := hex.DecodeString(magnet); err != nil {
			return "", ErrInvalidMagnet
		}
		return "magnet:?xt=urn:btih:" + magnet, nil
	}
	if !strings.HasPrefix(magnet, "magnet:?") {
		return "", ErrInvalidMagnet
	}

	parsed, err := url.Parse(magnet)
	if err != nil || parsed.Scheme != "magnet" {
		return "", ErrInvalidMagnet
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil || strings.TrimSpace(query.Get("xt")) == "" {
		return "", ErrInvalidMagnet
	}
	return magnet, nil
}
