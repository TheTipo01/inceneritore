package main

import "time"

// Config holds info for a server
type Config struct {
	ruolo    string
	testuale string
	vocale   string
	invito   string
	nome     string
	lastKick map[string]*time.Time
	messagge string
}
