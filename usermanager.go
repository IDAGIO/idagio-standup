package main

import (
	"github.com/nlopes/slack"
	"log"
	"sync"
)

type UserManager struct {
	client             *AuthedSlack
	messageReplies     chan slack.MessageEvent
	newStandups        chan newStandupForUser
	usersByUserId      map[string]*User
	usersByIMChannelId map[string]*User
	userIdBlacklist    map[string]bool
	channelIdBlacklist map[string]bool
	userListWaitMutex  sync.Mutex
}

type newStandupForUser struct {
	standup *Standup
	userId  string
	reply   chan bool
}

func NewUserManager(client *AuthedSlack) (um *UserManager) {
	um = &UserManager{
		client:             client,
		messageReplies:     make(chan slack.MessageEvent),
		newStandups:        make(chan newStandupForUser),
		usersByUserId:      make(map[string]*User),
		usersByIMChannelId: make(map[string]*User),
		userIdBlacklist:    make(map[string]bool),
		channelIdBlacklist: make(map[string]bool),
	}
	um.userListWaitMutex.Lock()
	go um.start()
	return
}

func (self *UserManager) StartStandup(s *Standup, userId string) (ok bool) {
	reply := make(chan bool, 1)
	self.newStandups <- newStandupForUser{standup: s, userId: userId,
		reply: reply}
	ok = <-reply
	return ok
}

func (self *UserManager) ReceiveMessageReply(m slack.MessageEvent) {
	self.messageReplies <- m
}

func (self *UserManager) start() {
	DebugLog.Println("UserManager started")

	var user *User
	var err error
	var ok bool

	for {
		user = nil

		select {
		case m := <-self.messageReplies:
			if user, ok = self.usersByIMChannelId[m.Channel]; !ok {
				user, err = self.lookupUserByIMChannelId(m.Channel)
				if err != nil {
					log.Printf(
						"error getting channel info; message dropped: %s", err)
					continue
				}
				if user == nil {
					continue
				} else {
					self.usersByUserId[user.Info.ID] = user
					self.usersByIMChannelId[m.Channel] = user
				}
			}
			DebugLog.Printf("delivering message %s to user %s", m.Timestamp, user.Info.Name)
			user.ReceiveMessageReply(m)

		case ns := <-self.newStandups:
			if user, ok = self.usersByUserId[ns.userId]; !ok {
				user, err = self.lookupUserById(ns.userId)
				if err != nil {
					log.Printf("error getting user info; new standup dropped: %s", err)
					ns.reply <- false
					continue
				}
				if user == nil {
					ns.reply <- false
					continue
				} else {
					self.usersByUserId[ns.userId] = user
					self.usersByIMChannelId[user.imChannelId] = user
				}
			}
			user.StartStandup(ns.standup)
			ns.reply <- true
		}
	}
}

func (self *UserManager) lookupUserByIMChannelId(channelId string) (user *User, err error) {
	if self.channelIdBlacklist[channelId] {
		return nil, nil
	}

	/* TODO: we could do better here, tracking opening and closing
	 * of IM channels with the RTM API
	 */
	ims, err := self.client.GetIMChannels()
	if err != nil {
		return
	}
	for _, im := range ims {
		if im.User == channelId {
			user, err = self.newUser(im.User, im.User)
			if user == nil && err == nil {
				self.channelIdBlacklist[channelId] = true
			}
			return
		}
	}
	return
}

func (self *UserManager) lookupUserById(userId string) (user *User, err error) {
	if self.userIdBlacklist[userId] {
		return nil, nil
	}

	_, _, channelId, err := self.client.OpenIMChannel(userId)
	if err != nil {
		return nil, err
	}
	return self.newUser(userId, channelId)
}

func (self *UserManager) newUser(userId string, imChannelId string) (user *User, err error) {
	userInfo, err := self.client.GetUserInfo(userId)
	if err != nil {
		return nil, err
	}
	if userInfo.IsBot {
		self.userIdBlacklist[userInfo.ID] = true
		return nil, nil
	}
	return NewUser(self.client, *userInfo, imChannelId), nil
}
