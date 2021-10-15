package model

import (
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/flosch/pongo2/v4"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Listen string
}

type SiteConfig struct {
	Title     string
	Path      string
	Theme     string
	ThemePath string
	Webroot   string
	MetaTags  []Metadata `yaml:"metaTags"`
	Users     map[string]string
}

func (c *SiteConfig) GetTemplatePath() string {
	return filepath.Join(c.ThemePath, "templates")
}

type Config struct {
	Server   ServerConfig
	Site     SiteConfig
	BasePath string
}

func NewConfig(conffilePath string) Config {
	config := Config{
		Server: ServerConfig{
			Listen: ":8080",
		},
		Site: SiteConfig{
			Theme: "default",
		},
	}

	log.Printf("Reading config file: %v", conffilePath)

	confFile, err := ioutil.ReadFile(conffilePath)
	if err != nil {
		log.Fatal(err)
	}

	config.BasePath = filepath.Dir(conffilePath)
	config.Site.Path, err = filepath.Abs(filepath.Join(config.BasePath, "site"))
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(confFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.Site.ThemePath == "" {
		absThemePath, err3 := filepath.Abs(filepath.Join(config.BasePath, "themes", config.Site.Theme))
		if err3 != nil {
			log.Fatal(err3)
		}
		config.Site.ThemePath = absThemePath
	}
	configPongoTemplatePathLoader(&config)

	return config
}

func configPongoTemplatePathLoader(conf *Config) {
	loader, err := pongo2.NewLocalFileSystemLoader(conf.Site.GetTemplatePath())
	if err != nil {
		log.Panic(err)
	}
	pongo2.DefaultSet.AddLoader(loader)
}
