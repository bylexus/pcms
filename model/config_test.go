package model

import (
	"runtime"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func loadTestConfig(t *testing.T, src string) (Config, error) {
	t.Helper()
	var c Config
	if err := yaml.Unmarshal([]byte(src), &c); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	return c, c.applyCacheAndResizerDefaults()
}

func TestCacheAndResizerDefaults(t *testing.T) {
	c, err := loadTestConfig(t, "server:\n  listen: \":3000\"\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := c.Server
	if s.CacheMaxAge == nil || *s.CacheMaxAge != DefaultCacheMaxAge {
		t.Errorf("CacheMaxAge: got %v, want %s", s.CacheMaxAge, DefaultCacheMaxAge)
	}
	if s.CacheCleanupInterval != DefaultCacheCleanupInterval {
		t.Errorf("CacheCleanupInterval: got %s, want %s", s.CacheCleanupInterval, DefaultCacheCleanupInterval)
	}
	r := s.ImageResizer
	if r.MaxWidth != DefaultImageMaxWidth || r.MaxHeight != DefaultImageMaxHeight {
		t.Errorf("max size: got %dx%d, want %dx%d", r.MaxWidth, r.MaxHeight, DefaultImageMaxWidth, DefaultImageMaxHeight)
	}
	if r.MaxConcurrent != runtime.NumCPU() {
		t.Errorf("MaxConcurrent: got %d, want %d", r.MaxConcurrent, runtime.NumCPU())
	}
	if r.WaitTimeout != DefaultImageWaitTimeout {
		t.Errorf("WaitTimeout: got %s, want %s", r.WaitTimeout, DefaultImageWaitTimeout)
	}
	if r.MaxSourceSize != DefaultImageMaxSourceSize {
		t.Errorf("MaxSourceSize: got %d, want %d", r.MaxSourceSize, DefaultImageMaxSourceSize)
	}
}

func TestCacheAndResizerExplicitValues(t *testing.T) {
	c, err := loadTestConfig(t, `
server:
  cache_max_age: "2h30m"
  cache_cleanup_interval: "5m"
  image_resizer:
    max_width: 800
    max_height: 600
    max_concurrent: 3
    wait_timeout: "500ms"
    max_source_size: 1048576
`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := c.Server
	if *s.CacheMaxAge != 150*time.Minute {
		t.Errorf("CacheMaxAge: got %s", *s.CacheMaxAge)
	}
	if s.CacheCleanupInterval != 5*time.Minute {
		t.Errorf("CacheCleanupInterval: got %s", s.CacheCleanupInterval)
	}
	r := s.ImageResizer
	if r.MaxWidth != 800 || r.MaxHeight != 600 || r.MaxConcurrent != 3 || r.WaitTimeout != 500*time.Millisecond || r.MaxSourceSize != 1048576 {
		t.Errorf("image_resizer: got %+v", r)
	}
}

func TestCacheMaxAgeZeroDisables(t *testing.T) {
	c, err := loadTestConfig(t, "server:\n  cache_max_age: \"0s\"\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Server.CacheMaxAge == nil || *c.Server.CacheMaxAge != 0 {
		t.Errorf("CacheMaxAge: got %v, want 0", c.Server.CacheMaxAge)
	}
}

func TestCacheMaxAgeBareIntRejected(t *testing.T) {
	var c Config
	if err := yaml.Unmarshal([]byte("server:\n  cache_max_age: 0\n"), &c); err == nil {
		t.Errorf("expected yaml error for bare integer duration, got nil")
	}
}

func TestCacheAndResizerNegativeValues(t *testing.T) {
	cases := map[string]string{
		"cache_max_age":          "server:\n  cache_max_age: \"-1h\"\n",
		"cache_cleanup_interval": "server:\n  cache_cleanup_interval: \"-1m\"\n",
		"max_width":              "server:\n  image_resizer:\n    max_width: -1\n",
		"max_height":             "server:\n  image_resizer:\n    max_height: -1\n",
		"max_concurrent":         "server:\n  image_resizer:\n    max_concurrent: -1\n",
		"wait_timeout":           "server:\n  image_resizer:\n    wait_timeout: \"-1s\"\n",
		"max_source_size":        "server:\n  image_resizer:\n    max_source_size: -1\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadTestConfig(t, src); err == nil {
				t.Errorf("expected error for negative %s, got nil", name)
			}
		})
	}
}
