package main

import "github.com/formicidae-tracker/dieu"

type Config struct {
	Emails []string
	Zones  map[string]dieu.Zone
}
