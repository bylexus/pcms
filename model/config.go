package model

import (
	"embed"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/flosch/pongo2/v4"
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

type Config struct {
	Server struct {
		Listen  string        `yaml:"listen"`
		Prefix  string        `yaml:"prefix"`
		Logging LoggingConfig `yaml:"logging"`
	} `yaml:"server"`
	Variables       map[string]interface{} `yaml:"variables"`
	ConfigFile      string
	SourcePath      string   `yaml:"source"`
	DestPath        string   `yaml:"dest"`
	TemplateDir     string   `yaml:"template_dir"`
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

func NewConfig(conffilePath string) Config {
	config := Config{}

	log.Printf("Reading config file: %v", conffilePath)

	// reading yaml config:
	confDir, _ := os.Getwd()
	confFile, err := os.ReadFile(conffilePath)
	if err != nil {
		log.Printf("Not using a config file - using default values\n")
	} else {
		confDir = path.Dir(conffilePath)
		err = yaml.Unmarshal(confFile, &config)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Serve mode
	if len(config.ServeMode) == 0 {
		config.ServeMode = SERVE_MODE_FILES
	}

	// set config defaults, if not set:
	if config.Server.Listen == "" {
		config.Server.Listen = ":8080"
	}

	// Set current working dir to the conf file dir for subsequent commands:
	err = os.Chdir(confDir)
	if err != nil {
		log.Fatal(err)
	}

	// source dir is relative to the working dir, or an absolute path:
	config.SourcePath, err = filepath.Abs(config.SourcePath)
	if err != nil {
		log.Fatal(err)
	}
	// dest dir is relative to the working dir, or an absolute path:
	config.DestPath, err = filepath.Abs(config.DestPath)
	if err != nil {
		log.Fatal(err)
	}

	// template dir is relative to the working dir, or an absolute path:
	config.TemplateDir, err = filepath.Abs(config.TemplateDir)
	if err != nil {
		log.Fatal(err)
	}

	configPongoTemplatePathLoader(config)

	return config
}

func configPongoTemplatePathLoader(conf Config) {
	loader, err := pongo2.NewLocalFileSystemLoader(conf.TemplateDir)
	if err != nil {
		log.Panic(err)
	}
	pongo2.DefaultSet.AddLoader(loader)
}
