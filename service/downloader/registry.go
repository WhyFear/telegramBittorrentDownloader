package downloader

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"telegramBittorrentDownloader/types"
)

// ErrDownloaderNotRegistered indicates that no factory exists for a downloader name.
var ErrDownloaderNotRegistered = errors.New("downloader is not registered")

type factory func(config types.Downloader) (Downloader, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]factory)
)

// NormalizeName returns the canonical key used to register and look up downloaders.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Register associates a downloader name with its constructor.
func Register(
	name string,
	constructor func(config types.Downloader) (Downloader, error),
) error {
	name = NormalizeName(name)
	if name == "" {
		return errors.New("downloader name cannot be empty")
	}
	if constructor == nil {
		return fmt.Errorf("downloader %q constructor cannot be nil", name)
	}

	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	if _, exists := factories[name]; exists {
		return fmt.Errorf("downloader %q is already registered", name)
	}
	factories[name] = constructor
	return nil
}

// IsRegistered reports whether a downloader factory has been registered for name.
func IsRegistered(name string) bool {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	_, exists := factories[NormalizeName(name)]
	return exists
}

// NewFromConfig creates a downloader using the factory selected by config.Name.
func NewFromConfig(config types.Downloader) (Downloader, error) {
	name := NormalizeName(config.Name)
	factoriesMu.RLock()
	constructor, exists := factories[name]
	factoriesMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrDownloaderNotRegistered, name)
	}

	dl, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize downloader %q: %w", name, err)
	}
	if dl == nil {
		return nil, fmt.Errorf("downloader %q constructor returned nil", name)
	}
	return dl, nil
}
