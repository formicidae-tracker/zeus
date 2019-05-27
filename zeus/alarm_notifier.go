package main

import "github.com/formicidae-tracker/zeus"

type AlarmNotifier interface {
	Notify(e zeus.AlarmEvent) error
}

type slackAlarmNotifier struct{}

// logs an alarm to a file
type logAlarmNotifier struct{}

// mails an alarm to some recipents
type mailAlarmNotifier struct{}

//Sends AlarmEvent to a website
type olympeAlarmNotifier struct{}
