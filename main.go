package main

import (
	"github.com/madebymany/tilly/Godeps/_workspace/src/github.com/abourget/slack"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
)

var Questions = []string{
	"What did you do yesterday?",
	"What are you planning to do today?",
	"Are you blocked by anything? If so, what?",
	"How are you feeling?",
}

const StandupTimeMinutes = 30

var StandupNagMinuteDelays = []int{15, 25}

const UserStandupStartText = "*WOOF!* Stand-up for #%s starting.\nMessage me `skip` to duck out of this one."
const UserStandupEndText = "Thanks! All done."
const UserStandupTimeUpText = "Too slow! The stand-up's finished now. Catch up in the channel."
const UserStandupAlreadyFinishedText = "Your next standup would have been for #%s but it's already finished. Catch up in the channel."
const UserNextStandupText = "But wait, you have another stand-up to attend…"
const UserConfirmSkipText = "Okay!"

var UserNagMessages = []string{
	"_nuzzle_ Don't forget me!",
	"_offers paw_ Do you have anything to say today?",
	"_wide puppy eyes_ Why are you so silent?",
	"Nudge. Nudgenudge.",
	"_stands right beside you, wagging tail so it thwocks against your leg_",
	"_paces around you_",
	"_drops the stand-up talking stick at your feet_",
}

var DefaultMessageParameters = slack.PostMessageParameters{
	AsUser:      true,
	Markdown:    true,
	Parse:       "full",
	EscapeText:  true,
	UnfurlLinks: true,
	UnfurlMedia: true,
	LinkNames:   1,
}

type AuthedSlack struct {
	*slack.Client
	UserId string
}

var DebugLog *log.Logger

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if os.Getenv("DEBUG") != "" {
		DebugLog = log.New(os.Stderr, "debug: ", log.LstdFlags|log.Lshortfile)
	} else {
		DebugLog = log.New(ioutil.Discard, "", 0)
	}
}

func main() {
	var err error

	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Fatalln("You must provide a SLACK_TOKEN environment variable")
	}

	client := slack.New(slackToken)

	auth, err := client.AuthTest()
	if err != nil {
		log.Fatalf("Couldn't log in: %s", err)
	}
	authClient := &AuthedSlack{Client: client, UserId: auth.UserId}

	slackWS := authClient.NewRTM()
	userManager := NewUserManager(authClient)
	eventReceiver := NewEventReceiver(slackWS, userManager, auth.UserId)
	go eventReceiver.Start()

	chs, err := authClient.GetChannels(true)
	if err != nil {
		log.Fatalf("Couldn't get channels: %s", err)
	}

	exitWaitGroup := new(sync.WaitGroup)

	for _, ch := range chs {
		if ch.IsGeneral || !ch.IsMember {
			continue
		}

		s := NewStandup(authClient, ch, userManager, exitWaitGroup)
		go s.Run()
	}

	exitWaitGroup.Wait()
}

func RandomisedNags() (out []string) {
	out = make([]string, len(UserNagMessages))
	copy(out, UserNagMessages)
	for i := range out {
		j := rand.Intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return
}
