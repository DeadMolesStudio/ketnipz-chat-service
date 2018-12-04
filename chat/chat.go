package chat

import (
	"sync"

	"github.com/gorilla/websocket"

	uuid "github.com/satori/go.uuid"

	db "github.com/go-park-mail-ru/2018_2_DeadMolesStudio/database"
	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"

	"chat/database"
	"chat/models"
)

var chat *Chat

type Chat struct {
	Users *sync.Map

	dm *db.DatabaseManager

	Join  chan *User
	Leave chan *User
	Send  chan *ProcessWSMessage
}

//easyjson:json
type WSMessageToSend struct {
	Action  string      `json:"action"`
	Payload interface{} `json:"payload"`
}

func (c *Chat) Run() {
	go c.acceptJoiningUsers()
	go c.acceptLeavingUsers()
	go c.acceptSendingMessages()
}

func (c *Chat) acceptJoiningUsers() {
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

func (c *Chat) acceptLeavingUsers() {
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

func (c *Chat) acceptSendingMessages() {
	for {
		m := <-c.Send
		wsm := m.WSM.(*ReceivedWSMessage)
		if !m.From.Anon {
			logger.Infof("Got message from %v: action = %v, payload = %v", m.From.UID, wsm.Action, string(wsm.Payload))
		} else {
			logger.Infof("Got message from %v: action = %v, payload = %v", m.From.SessionID, wsm.Action, string(wsm.Payload))
		}
		switch wsm.Action {
		case "get":
			c.getAllMessages(m)
		case "send":
			c.sendMessage(m)
		default:
			c.sendWSMessageToSession(&ProcessWSMessage{
				From: m.From,
				WSM: &WSMessageToSend{
					Action:  "error",
					Payload: "unknown action type",
				},
			})
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
	wsm := m.WSM.(*WSMessageToSend)
	j, err := wsm.MarshalJSON()
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("sending the message: %v", string(j))
	err = d.Conn.WriteMessage(websocket.TextMessage, j)
	if err != nil {
		logger.Infof("Error while sending to user %v: %v", *d, err)
	}
}

func (c *Chat) getAllMessages(m *ProcessWSMessage) {
	res, err := database.GetAllGlobalMessages(c.dm)
	logger.Infof("gotcha all messages, request from %v", m.From.SessionID)
	if err != nil {
		logger.Error(err)
		return
	}
	c.sendWSMessageToSession(&ProcessWSMessage{
		From: m.From,
		WSM: &WSMessageToSend{
			Action: "get",
			Payload: models.Messages{
				Msgs: res,
			},
		},
	})
}

func (c *Chat) sendMessage(m *ProcessWSMessage) {
	wsm := m.WSM.(*ReceivedWSMessage)
	mess := &models.Message{}
	err := mess.UnmarshalJSON(wsm.Payload)
	if err != nil {
		logger.Infof("Message cannot be parsed: %v, message: %v", err, wsm.Payload)
		c.sendWSMessageToSession(&ProcessWSMessage{
			From: m.From,
			WSM: &WSMessageToSend{
				Action:  "error",
				Payload: "bad payload",
			},
		})
		return
	}

	if !m.From.Anon {
		mess.Author = new(uint)
		*mess.Author = m.From.UID
	}
	mess, err = database.CreateMessage(c.dm, mess)
	if err != nil {
		logger.Infof("Message cannot be saved: %v", err)
		return
	}
	logger.Infof("Message saved to database: %v", *mess)

	send := &WSMessageToSend{
		Action:  "send",
		Payload: mess,
	}
	j, err := send.MarshalJSON()
	if err != nil {
		logger.Error("Marshalling ended with error: %v", err)
		return
	}
	logger.Debugf("sending the message: %v", string(j))
	if mess.To == nil {
		if mess.Author != nil {
			logger.Infof("Got global message from %v: %v", *mess.Author, mess.Message)
		} else {
			logger.Info("Got global message from anonym: ", mess.Message)
		}
		c.Users.Range(func(k, v interface{}) bool {
			d := v.(*Data)
			err = d.Conn.WriteMessage(websocket.TextMessage, j)
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
					err = d.Conn.WriteMessage(websocket.TextMessage, j)
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
				c.sendWSMessageToSession(&ProcessWSMessage{
					From: m.From,
					WSM: &WSMessageToSend{
						Action:  "send",
						Payload: "user if offline",
					},
				})
			}
		} else {
			logger.Info("anonymous users can't send private messages")
		}
	}
}

func InitChat(dm *db.DatabaseManager) *Chat {
	chat = &Chat{
		Users: &sync.Map{},
		dm:    dm,
		Join:  make(chan *User),
		Leave: make(chan *User),
		Send:  make(chan *ProcessWSMessage),
	}
	return chat
}

func JoinUser(u *User) {
	chat.Join <- u
}
