package main

// Server holds info for a server
type Server struct {
	ruolo     string
	testuale  string
	vocale    string
	invito    string
	nome      string
	messagge  string
	boostRole string
}

type Config struct {
	Token    string `fig:"token" validate:"required"`
	DSN      string `fig:"datasourcename" validate:"required"`
	Driver   string `fig:"drivername" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
	Site     string `fig:"site" validate:"required"`
}
