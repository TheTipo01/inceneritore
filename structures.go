package main

// Server holds info for a server
type Server struct {
	ServerID            string   `fig:"serverID" validate:"required"`
	ServerName          string   `fig:"serverName" validate:"required"`
	Role                string   `fig:"role"`
	TextChannel         string   `fig:"textChannel"`
	VoiceChannel        string   `fig:"voiceChannel"`
	Invite              string   `fig:"invite"`
	Message             string   `fig:"message"`
	BoostRole           string   `fig:"boostRole"`
	LockdownChannel     string   `fig:"lockdownChannel"`
	LockdownBlacklisted []string `fig:"lockdownBlacklisted"`
	BlacklistMap        map[string]struct{}
}

type Config struct {
	Token    string   `fig:"token" validate:"required"`
	DSN      string   `fig:"datasourcename" validate:"required"`
	Driver   string   `fig:"drivername" validate:"required"`
	LogLevel string   `fig:"loglevel" validate:"required"`
	Site     string   `fig:"site" validate:"required"`
	Server   []Server `fig:"server" validate:"required"`
}
