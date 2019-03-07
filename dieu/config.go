package main

import "git.tuleu.science/fort/dieu"

type Config struct {
	Emails []string
	Zones  map[string]dieu.Zone
}
