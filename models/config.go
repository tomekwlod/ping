package models

type DBConfig struct {
	Database string
	Addr     string
	Port     string
}
type SMTPConfig struct {
	Email    string
	Password string
	Server   string
	Port     string
	Emails   []string
}
