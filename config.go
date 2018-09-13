package ping

import (
	"os"

	"github.com/jinzhu/configor"
)

type Parameters struct {
	SMTP_Email    string
	SMTP_Password string
	SMTP_Server   string
	SMTP_Port     string
	SMTP_Emails   []string
}

func LoadConfig() (p Parameters, err error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// or Panic and env should be everytime present, even on dev
		configPath = "../../configs"
	}

	p = Parameters{}
	err = configor.Load(&p, configPath+"/parameters.yml")
	if err != nil {
		return
	}

	return
}
