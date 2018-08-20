package ping

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
