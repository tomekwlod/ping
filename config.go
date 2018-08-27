package ping

import (
	"os"

	"github.com/jinzhu/configor"
)

type Parameters struct {
	GO_Port string

	MongoDB_Database string
	MongoDB_Addr     string
	MongoDB_Port     string

	SMTP_Email    string
	SMTP_Password string
	SMTP_Server   string
	SMTP_Port     string
	SMTP_Emails   []string
}

func LoadConfig() Parameters {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// or Panic and env should be everytime present, even on dev
		configPath = "../../configs"
	}

	p := Parameters{}
	if err := configor.Load(&p, configPath+"/parameters.yml"); err != nil {
		panic(err)
	}

	return p
}
