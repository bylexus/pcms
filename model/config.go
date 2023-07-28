package model

import (
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/flosch/pongo2/v4"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Listen  string        `yaml:"listen"`
	Logging LoggingConfig `yaml:"logging"`
}

type LoggingConfig struct {
	Access LoggingConfigEntry `yaml:"access"`
	Error  LoggingConfigEntry `yaml:"error"`
}

type LoggingConfigEntry struct {
	File   string `yaml:"file"`
	Format string `yaml:"format"`
	Level  string `yaml:"level"`
}

type Config struct {
	Server          ServerConfig           `yaml:"server"`
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
}

func NewConfig(conffilePath string) Config {
	config := Config{
		Server: ServerConfig{
			Listen: ":8080",
		},
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

	// config.Site.Path, err = filepath.Abs(filepath.Join(config.BasePath, "site"))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if config.Site.ThemePath == "" {
	// 	absThemePath, err3 := filepath.Abs(filepath.Join(config.BasePath, "themes", config.Site.Theme))
	// 	if err3 != nil {
	// 		log.Fatal(err3)
	// 	}
	// 	config.Site.ThemePath = absThemePath
	// }
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
