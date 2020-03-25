package main

import (
	"github.com/nlopes/slack"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

var Questions = []string{
	"What did you do yesterday? ⬅️ ",
	"What are you planning to do today? ➡️ ",
	"Are you blocked by anything? If so, what? ⛔️ ",
	"How are you feeling? 🏖 ",
}

const StandupTimeMinutes = 210

var StandupNagMinuteDelays = []int{195, 205}

const UserStandupStartText = "*WOOF!* Stand-up for #%s starting.\nMessage me `skip` to duck out of this one."
const UserStandupEndText = "Thanks! 🙏 All done."
const UserStandupTimeUpText = "Too slow! The stand-up's finished now. Catch up in the channel."
const UserStandupAlreadyFinishedText = "Your next standup would have been for #%s but it's already finished. Catch up in the channel."
const UserNextStandupText = "But wait, you have another stand-up to attend…"
const UserConfirmSkipText = "Okay!"

var UserNagMessages = []string{
	"Don't forget to answer me!",
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

	if day := time.Now().Weekday(); os.Getenv("TILLY_WEEKDAY_ONLY") != "" && (day < time.Monday || day > time.Friday) {
		log.Fatalln("Exiting; it's the weekend and I'm set to only run on a weekday")
	}

	client := slack.New(slackToken)

	auth, err := client.AuthTest()
	if err != nil {
		log.Fatalf("Couldn't log in: %s", err)
	}
	authClient := &AuthedSlack{Client: client, UserId: auth.UserID}

	slackWS := authClient.NewRTM()
	userManager := NewUserManager(authClient)
	eventReceiver := NewEventReceiver(slackWS, userManager, auth.UserID)
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

		channelInfo, err := authClient.GetChannelInfo(ch.ID)
		if err != nil {
			log.Fatalf("Couldn't get channel info: %s", err)
		}

		DebugLog.Print(len(channelInfo.Members))
		s := NewStandup(authClient, channelInfo, userManager, exitWaitGroup)
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
