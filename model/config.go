package model

import (
	"embed"
	"fmt"
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

func NewConfig(conffilePath string, cliArgs CmdArgs) Config {
	config := Config{}
	config.ConfigFile = conffilePath

	// determine mode:
	switch cliArgs.FlagSet.Name() {
	case "serve":
		config.ServeMode = SERVE_MODE_FILES
	case "serve-doc":
		config.ServeMode = SERVE_MODE_EMBEDDED_DOC
	case "init":
		// we don't need to parse the config in init mode:
		return config
	}

	// read command specific flags
	if listen := cliArgs.FlagSet.Lookup("listen"); listen != nil {
		config.Server.Listen = listen.Value.String()
	}

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

	// set config defaults, if not set:
	if config.Server.Listen == "" {
		config.Server.Listen = ":8080"
	}

	// Set current working dir to the conf file dir for subsequent commands:
	err = os.Chdir(confDir)
	if err != nil {
		log.Fatal(err)
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
		// dest dir is relative to the working dir, or an absolute path:
		if len(config.DestPath) == 0 {
			log.Fatal(fmt.Errorf("SourcePath cannot be empty"))
		}
		config.DestPath, err = filepath.Abs(config.DestPath)
		if err != nil {
			log.Fatal(err)
		}

		// source and dest path must not be the same
		if config.SourcePath == config.DestPath {
			log.Fatal("Source and Destination path must not be the same")
		}

		// source and dest path must not be the same as the actual cwd:
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		if config.SourcePath == cwd || config.DestPath == cwd {
			log.Fatal("Source and Destination path must not be in the actual working dir")
		}

		// template dir is relative to the working dir, or an absolute path:
		config.TemplateDir, err = filepath.Abs(config.TemplateDir)
		if err != nil {
			log.Fatal(err)
		}
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
