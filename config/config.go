package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

var Params struct {
	// URLsFile string
	DB struct {
		DbName  string
		Address string
		Port    string
	}
	SMTP struct {
		Email    string
		Password string
		Server   string
		Port     string
		Emails   []string
	}
}

// var tomlFile = "parameters.toml"

// var (
// 	_, b, _, _ = runtime.Caller(0)
// 	basepath   = filepath.Dir(b)
// )

func init() {
	if _, err := toml.DecodeFile("config/parameters.toml", &Params); err != nil {
		log.Fatal(err)
	}
}
