package main

import "time"

// Server holds info for a server
type Server struct {
	ruolo    string
	testuale string
	vocale   string
	invito   string
	nome     string
	lastKick map[string]*time.Time
	messagge string
}
