package model

import "path/filepath"

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
