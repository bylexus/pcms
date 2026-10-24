package model

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/flosch/pongo2/v6"
	"gopkg.in/yaml.v3"
)

type LoggingConfig struct {
	Access LoggingConfigEntry `yaml:"access"`
	Error  LoggingConfigEntry `yaml:"error"`
}

type LoggingConfigEntry struct {
	File   string `yaml:"file"`
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

const (
	SERVE_MODE_FILES        = "FILES"
	SERVE_MODE_EMBEDDED_DOC = "EMBEDDED_DOC"
)

// ImageResizerConfig holds the limits for the /_imageResizer/ endpoint.
type ImageResizerConfig struct {
	// Maximum output width / height in pixels. Also the upper limit for the
	// width/height/maxWidth/maxHeight request parameters.
	MaxWidth  int `yaml:"max_width"`
	MaxHeight int `yaml:"max_height"`
	// Maximum number of uncached resize operations running at the same time.
	// 0 means: number of CPUs.
	MaxConcurrent int `yaml:"max_concurrent"`
	// How long a request waits for a free resize slot before it gets a 503.
	WaitTimeout time.Duration `yaml:"wait_timeout"`
	// Maximum source image file size in bytes. Larger images get a 413.
	MaxSourceSize int64 `yaml:"max_source_size"`
}

const (
	DefaultImageMaxSourceSize   = 32 * 1024 * 1024 // 32 MB
	DefaultImageMaxWidth        = 1920
	DefaultImageMaxHeight       = 1920
	DefaultImageWaitTimeout     = 10 * time.Second
	DefaultCacheMaxAge          = 24 * time.Hour
	DefaultCacheCleanupInterval = time.Hour
)

type Config struct {
	Server struct {
		Listen   string `yaml:"listen"`
		Watch    bool   `yaml:"watch"`
		Prefix   string `yaml:"prefix"`
		CacheDir string `yaml:"cache_dir"`
		// Cache files not used for this duration are removed by the cleanup job.
		// nil (not set) means the default; 0 disables the cleanup.
		CacheMaxAge          *time.Duration     `yaml:"cache_max_age"`
		CacheCleanupInterval time.Duration      `yaml:"cache_cleanup_interval"`
		ImageResizer         ImageResizerConfig `yaml:"image_resizer"`
		Logging              LoggingConfig      `yaml:"logging"`
	} `yaml:"server"`
	Variables       map[string]interface{} `yaml:"variables"`
	ConfigFile      string
	SourcePath      string   `yaml:"source"`
	TemplateDir     string   `yaml:"template_dir"`
	DatabasePath    string   `yaml:"database_path"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
	Processors      struct {
		Html struct{} `yaml:"html"`
		Scss struct {
			SassBin string `yaml:"sass_bin"`
		} `yaml:"scss"`
	} `yaml:"processors"`
	EmbeddedDocFS embed.FS
	ServeMode     string
}

func NewConfig(conffilePath string, cliArgs CmdArgs, embeddedDocFS embed.FS) Config {
	config := Config{}
	config.ConfigFile = conffilePath
	config.EmbeddedDocFS = embeddedDocFS
	serveMode := ""

	// determine mode:
	switch cliArgs.FlagSet.Name() {
	case "serve":
		serveMode = SERVE_MODE_FILES
	case "serve-doc":
		serveMode = SERVE_MODE_EMBEDDED_DOC
	case "index":
		serveMode = SERVE_MODE_FILES
		// config.ServeMode = SERVE_MODE_EMBEDDED_DOC
	case "init":
		// we don't need to parse the config in init mode:
		return config
	}
	config.ServeMode = serveMode

	log.Printf("Reading config file: %v", conffilePath)

	// reading yaml config:
	confDir, _ := os.Getwd()
	var (
		confFile []byte
		err      error
	)

	if serveMode == SERVE_MODE_EMBEDDED_DOC {
		confFile, err = config.EmbeddedDocFS.ReadFile(conffilePath)
		if err != nil {
			log.Fatal(fmt.Errorf("embedded doc config file not found (%s): %w", conffilePath, err))
		}
		confDir = path.Dir(conffilePath)
	} else {
		confFile, err = os.ReadFile(conffilePath)
		if err != nil {
			log.Printf("Not using a config file - using default values\n")
		} else {
			confDir = path.Dir(conffilePath)
		}
	}

	if len(confFile) > 0 {
		err = yaml.Unmarshal(confFile, &config)
		if err != nil {
			log.Fatal(err)
		}
		config.ServeMode = serveMode
		config.ConfigFile = conffilePath
		config.EmbeddedDocFS = embeddedDocFS
	}

	// read command specific flags
	if listen := cliArgs.FlagSet.Lookup("listen"); listen != nil {
		config.Server.Listen = listen.Value.String()
	}

	// set config defaults, if not set:
	if config.Server.Listen == "" {
		config.Server.Listen = ":8080"
	}
	if config.Server.CacheDir == "" {
		config.Server.CacheDir = ".pcms-cache"
	}
	if config.DatabasePath == "" {
		config.DatabasePath = "pcms.db"
	}
	if err := config.applyCacheAndResizerDefaults(); err != nil {
		log.Fatal(fmt.Errorf("invalid config: %w", err))
	}

	// Set current working dir to the conf file dir for subsequent commands,
	// except when serving embedded docs.
	if config.ServeMode != SERVE_MODE_EMBEDDED_DOC {
		err = os.Chdir(confDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	if config.ServeMode != SERVE_MODE_EMBEDDED_DOC {
		// source dir is relative to the working dir, or an absolute path:
		if len(config.SourcePath) == 0 {
			log.Fatal(fmt.Errorf("SourcePath cannot be empty"))
		}
		config.SourcePath, err = filepath.Abs(config.SourcePath)
		if err != nil {
			log.Fatal(err)
		}
		config.DatabasePath, err = filepath.Abs(config.DatabasePath)
		if err != nil {
			log.Fatal(err)
		}
		if cliArgs.FlagSet.Name() == "serve" {
			// template dir is relative to the working dir, or an absolute path:
			config.TemplateDir, err = filepath.Abs(config.TemplateDir)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if cliArgs.FlagSet.Name() == "serve" {
		config.Server.CacheDir, err = filepath.Abs(config.Server.CacheDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	if cliArgs.FlagSet.Name() == "serve" || cliArgs.FlagSet.Name() == "serve-doc" {
		configPongoTemplatePathLoader(config)
	}

	return config
}

// applyCacheAndResizerDefaults sets the defaults for the cache cleanup and
// image resizer settings, and rejects negative values.
func (config *Config) applyCacheAndResizerDefaults() error {
	s := &config.Server
	r := &s.ImageResizer

	if s.CacheMaxAge == nil {
		d := DefaultCacheMaxAge
		s.CacheMaxAge = &d
	}
	if *s.CacheMaxAge < 0 {
		return fmt.Errorf("server.cache_max_age must not be negative, got %s", *s.CacheMaxAge)
	}
	if s.CacheCleanupInterval < 0 {
		return fmt.Errorf("server.cache_cleanup_interval must not be negative, got %s", s.CacheCleanupInterval)
	}
	if s.CacheCleanupInterval == 0 {
		s.CacheCleanupInterval = DefaultCacheCleanupInterval
	}

	if r.MaxWidth < 0 {
		return fmt.Errorf("server.image_resizer.max_width must not be negative, got %d", r.MaxWidth)
	}
	if r.MaxWidth == 0 {
		r.MaxWidth = DefaultImageMaxWidth
	}
	if r.MaxHeight < 0 {
		return fmt.Errorf("server.image_resizer.max_height must not be negative, got %d", r.MaxHeight)
	}
	if r.MaxHeight == 0 {
		r.MaxHeight = DefaultImageMaxHeight
	}
	if r.MaxConcurrent < 0 {
		return fmt.Errorf("server.image_resizer.max_concurrent must not be negative, got %d", r.MaxConcurrent)
	}
	if r.MaxConcurrent == 0 {
		r.MaxConcurrent = runtime.NumCPU()
	}
	if r.WaitTimeout < 0 {
		return fmt.Errorf("server.image_resizer.wait_timeout must not be negative, got %s", r.WaitTimeout)
	}
	if r.WaitTimeout == 0 {
		r.WaitTimeout = DefaultImageWaitTimeout
	}
	if r.MaxSourceSize < 0 {
		return fmt.Errorf("server.image_resizer.max_source_size must not be negative, got %d", r.MaxSourceSize)
	}
	if r.MaxSourceSize == 0 {
		r.MaxSourceSize = DefaultImageMaxSourceSize
	}
	return nil
}

func configPongoTemplatePathLoader(conf Config) {
	if conf.ServeMode == SERVE_MODE_EMBEDDED_DOC {
		templateRoot := path.Clean(path.Join(path.Dir(conf.ConfigFile), conf.TemplateDir))
		loader, err := pongo2.NewHttpFileSystemLoader(http.FS(conf.EmbeddedDocFS), templateRoot)
		if err != nil {
			log.Fatal(fmt.Errorf("configure embedded pongo2 loader (%s): %w", templateRoot, err))
		}
		pongo2.DefaultSet.AddLoader(loader)
		return
	}

	loader, err := pongo2.NewLocalFileSystemLoader(conf.TemplateDir)
	if err != nil {
		log.Fatal(fmt.Errorf("configure local pongo2 loader (%s): %w", conf.TemplateDir, err))
	}
	pongo2.DefaultSet.AddLoader(loader)
}

func GetEmbeddedSourceFS(config Config) (fs.FS, string, error) {
	if config.SourcePath == "" {
		return nil, "", fmt.Errorf("source path empty")
	}

	embeddedRoot := path.Clean(path.Join(path.Dir(config.ConfigFile), config.SourcePath))
	subFS, err := fs.Sub(config.EmbeddedDocFS, embeddedRoot)
	if err != nil {
		return nil, "", fmt.Errorf("create embedded source fs (%s): %w", embeddedRoot, err)
	}

	return subFS, "embedded:" + embeddedRoot, nil
}
