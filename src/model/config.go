package model

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
}

type Config struct {
	Server   ServerConfig
	Site     SiteConfig
	BasePath string
}
