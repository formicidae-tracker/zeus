package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/formicidae-tracker/zeus"
	"github.com/slack-go/slack"
)

type slackReporter struct {
	c        *slack.Client
	userID   string
	zoneName string
	hostName string
	events   chan zeus.AlarmEvent
}

func (r *slackReporter) formatEvent(e zeus.AlarmEvent) string {
	icon := ":ok:"
	alarmText := "alarm is off."
	if e.Status == zeus.AlarmOn {
		icon = ":warning:"
		alarmText = "alarm is on!"
	}
	return fmt.Sprintf("%s %s.%s : '%s' %s", icon, r.hostName, e.Zone, e.Reason, alarmText)
}

func (r *slackReporter) Report() {
	r.c.PostMessage(r.userID, slack.MsgOptionText(fmt.Sprintf(":ok: climate control on %s.%s started.", r.hostName, r.zoneName), true))
	for e := range r.events {
		if e.Flags&zeus.InstantNotification == 0 {
			continue
		}
		if e.Zone != path.Join(r.hostName, "zone", r.zoneName) {
			continue
		}

		_, _, err := r.c.PostMessage(r.userID, slack.MsgOptionText(r.formatEvent(e), true))
		if err != nil {
			fmt.Fprintf(os.Stderr, "[zone/%s/slack] cannot notify alarm: %s\n", r.zoneName, err)
		}
	}
	r.c.PostMessage(r.userID, slack.MsgOptionText(fmt.Sprintf(":ok: climate control on %s.%s stopped.", r.hostName, r.zoneName), true))
}

func (r *slackReporter) AlarmChannel() chan<- zeus.AlarmEvent {
	return r.events
}

func FindSlackUser(c *slack.Client, username string) (string, error) {
	username = strings.TrimPrefix(username, "@")
	users, err := c.GetUsers()
	if err != nil {
		return "", err
	}
	for _, user := range users {
		if user.Profile.DisplayName == username {
			return user.ID, nil
		}
	}
	return "", fmt.Errorf("Could not find user @%s on slack", username)
}

func NewSlackReporter(c *slack.Client, userID, zoneName string) (AlarmReporter, error) {
	res := &slackReporter{
		c:        c,
		userID:   userID,
		zoneName: zoneName,
		events:   make(chan zeus.AlarmEvent),
	}
	var err error
	res.hostName, err = os.Hostname()
	if err != nil {
		return nil, err
	}
	return res, nil
}
