package bili

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultUserAgent is a realistic desktop Chrome UA. Bilibili rejects unusual
// or empty agents with code -412.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Referer is required on most endpoints to avoid interception.
const Referer = "https://www.bilibili.com"

// Config holds everything the client needs. The zero value is usable after
// passing through DefaultConfig.
type Config struct {
	Cookie    string
	UserAgent string
	Proxy     string
	Lang      string

	Rate    time.Duration
	Retries int
	Timeout time.Duration

	CacheDir string
	CacheTTL time.Duration
	NoCache  bool

	// DryRun makes every request print its method and URL to DryRunWriter
	// (default os.Stdout) and return an empty success envelope instead of
	// hitting the network.
	DryRun bool
}

// DefaultConfig returns sane, polite defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Lang:      "zh-CN",
		Rate:      350 * time.Millisecond,
		Retries:   4,
		Timeout:   30 * time.Second,
		CacheDir:  cacheDir(),
		CacheTTL:  time.Hour,
	}
}

func cacheDir() string {
	if d := os.Getenv("BILI_CACHE_DIR"); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "bili")
	}
	return filepath.Join(os.TempDir(), "bili-cache")
}

// ConfigDir returns the directory for the config file.
func ConfigDir() string {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "bili")
	}
	return filepath.Join(os.TempDir(), "bili-config")
}

// DataDir returns the directory for the local database default.
func DataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "bili")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "bili")
	}
	return filepath.Join(os.TempDir(), "bili-data")
}
