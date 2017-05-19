package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

var Params struct {
	// URLsFile string
	SMTP struct {
		Email    string
		Password string
		Server   string
		Port     string
	}
	Emails []string
}

var tomlFile = "config/parameters.toml"

func init() {
	if _, err := toml.DecodeFile(tomlFile, &Params); err != nil {
		log.Fatal(err)
	}
}
