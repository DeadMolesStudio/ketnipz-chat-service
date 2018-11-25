package chat

import (
	"sync"

	uuid "github.com/satori/go.uuid"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"

	"chat/models"
)

var chat *Chat

type Chat struct {
	Users *sync.Map

	Join  chan *User
	Leave chan *User
	Send  chan *ProcessWSMessage
}

func (c *Chat) Run() {
	go c.AcceptJoiningUsers()
	go c.AcceptLeavingUsers()
	go c.AcceptSendingMessages()
}

func (c *Chat) AcceptJoiningUsers() {
	for {
		u := <-c.Join
		if u.SessionID == "" {
			u.SessionID = uuid.NewV4().String()
		}
		c.Users.Store(u.SessionID, &u.Data)
		go u.Listen()
		if !u.Anon {
			logger.Infof("User with id %v joined chat with session %v", u.UID, u.SessionID)
		} else {
			logger.Infof("anonymous user joined chat with temp session %v", u.SessionID)
		}
	}
}

func (c *Chat) AcceptLeavingUsers() {
	for {
		u := <-c.Leave
		c.Users.Delete(u.SessionID)
		if !u.Anon {
			logger.Infof("User with id %v left chat with session %v", u.UID, u.SessionID)
		} else {
			logger.Infof("anonymous user left chat with temp session %v", u.SessionID)
		}
	}
}

func (c *Chat) AcceptSendingMessages() {
	for {
		m := <-c.Send
		if !m.From.Anon {
			logger.Infof("Got message from %v: action = %v, payload = %v", m.From.UID, m.WSM.Action, m.WSM.Payload)
		} else {
			logger.Infof("Got message from %v: action = %v, payload = %v", m.From.SessionID, m.WSM.Action, m.WSM.Payload)
		}
		switch m.WSM.Action {
		case "get":
			c.getAllMessages(m)
		case "send":
			c.sendMessage(m)
		case "error":
			c.sendWSMessageToSession(m)
		}
	}
}

func (c *Chat) sendWSMessageToSession(m *ProcessWSMessage) {
	u, ok := c.Users.Load(m.From.SessionID)
	if !ok {
		logger.Info("user cannot be found")
		return
	}
	d := u.(*Data)
	err := d.Conn.WriteJSON(m.WSM)
	if err != nil {
		logger.Infof("Error while sending to user %v: %v", *d, err)
	}
}

func (c *Chat) getAllMessages(m *ProcessWSMessage) {
	c.sendWSMessageToSession(m)
}

func (c *Chat) sendMessage(m *ProcessWSMessage) {
	mess := m.WSM.Payload.(*models.Message)
	if mess.To == nil {
		if mess.Author != nil {
			logger.Infof("Got global message from %v: %v", *mess.Author, mess.Message)
		} else {
			logger.Info("Got global message from anonym: ", mess.Message)
		}
		c.Users.Range(func(k, v interface{}) bool {
			d := v.(*Data)
			err := d.Conn.WriteJSON(m.WSM)
			if err != nil {
				logger.Info(err)
			}
			if mess.Author != nil {
				logger.Infof("Message sent from %v: %v", *mess.Author, mess.Message)
			} else {
				logger.Infof("Message sent from anonym: %v", mess.Message)
			}
			return true
		})
	} else {
		if mess.Author != nil {
			sent := false
			logger.Info("Got private message from %v to %v: %v", *mess.Author, *mess.To, mess.Message)
			c.Users.Range(func(k, v interface{}) bool {
				d := v.(*Data)
				if d.UID == *mess.To {
					err := d.Conn.WriteJSON(m.WSM)
					if err != nil {
						logger.Info(err)
						return false
					}
					sent = true
					logger.Info("Private message from %v to %v: %v", *mess.Author, *mess.To, mess.Message)
					return false
				}
				return true
			})
			if !sent {
				logger.Info("user %v is offline", *mess.To)
				m.WSM = &SendWSMessage{
					Action:  "send",
					Payload: "user is offline",
				}
				c.sendWSMessageToSession(m)
			}
		} else {
			logger.Info("anonymous users can't send private messages")
		}
	}
}

func InitChat() *Chat {
	chat = &Chat{
		Users: &sync.Map{},
		Join:  make(chan *User),
		Leave: make(chan *User),
		Send:  make(chan *ProcessWSMessage),
	}
	return chat
}

func JoinUser(u *User) {
	chat.Join <- u
}
