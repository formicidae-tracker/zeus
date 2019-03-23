package main

import "github.com/formicidae-tracker/dieu"

type AlarmNotifier interface {
	Notify(e dieu.AlarmEvent) error
}

type slackAlarmNotifier struct{}

// logs an alarm to a file
type logAlarmNotifier struct{}

// mails an alarm to some recipents
type mailAlarmNotifier struct{}

//Sends AlarmEvent to a website
type olympeAlarmNotifier struct{}
