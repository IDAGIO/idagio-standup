package main

import (
	"github.com/nlopes/slack"
)

type EventReceiver struct {
	client      *slack.RTM
	userManager *UserManager
	botUserId   string
}

func NewEventReceiver(client *slack.RTM, um *UserManager, botUserId string) (er *EventReceiver) {
	client.IncomingEvents = make(chan slack.RTMEvent)
	return &EventReceiver{
		client:      client,
		userManager: um,
		botUserId:   botUserId,
	}
}

func (self *EventReceiver) Start() {
	go self.client.ManageConnection()
	DebugLog.Println("EventReceiver started")
	for ev := range self.client.IncomingEvents {
		if m, ok := ev.Data.(*slack.MessageEvent); ok && m.Msg.User != self.botUserId && m.Msg.Text != "" {
			DebugLog.Printf("Received message id %s from RTM, userId '%s' : %s", m.Timestamp, m.Msg.User, m.Msg.Text)
			self.userManager.ReceiveMessageReply(*m)
		}
	}
}
