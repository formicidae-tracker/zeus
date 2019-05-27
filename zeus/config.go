package main

import "github.com/formicidae-tracker/zeus"

type Config struct {
	Emails []string
	Zones  map[string]zeus.Zone
}
