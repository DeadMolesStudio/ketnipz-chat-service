package chat

import (
	"encoding/json"

	"github.com/go-park-mail-ru/2018_2_DeadMolesStudio/logger"
	"github.com/gorilla/websocket"

	"chat/database"
	"chat/models"
)

type User struct {
	SessionID string
	Data
}

type Data struct {
	UID  uint
	Conn *websocket.Conn
	Anon bool
}

type ProcessWSMessage struct {
	WSM  *SendWSMessage
	From *User
}

//easyjson:json
type WSMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

//easyjson:json
type SendWSMessage struct {
	Action  string      `json:"action"`
	Payload interface{} `json:"payload"`
}

func (u *User) Listen() {
	for {
		m := &WSMessage{}
		_, raw, err := u.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				if !u.Anon {
					logger.Infof("User %v with session %v was disconnected", u.UID, u.SessionID)
				} else {
					logger.Infof("anonymous user with temp session %v was disconnected", u.SessionID)
				}
			} else {
				logger.Error(err)
			}
			chat.Leave <- u
			return
		}
		err = m.UnmarshalJSON(raw)
		if err != nil {
			logger.Error(err)
			continue
		}

		logger.Infof("Read WSMessage: %v", *m)

		switch m.Action {
		case "get":
			res, err := database.GetAllGlobalMessages()
			logger.Infof("gotcha all messages, request from %v", u.SessionID)
			if err != nil {
				logger.Error(err)
				continue
			}
			chat.Send <- &ProcessWSMessage{
				From: u,
				WSM: &SendWSMessage{
					Action: "get",
					Payload: models.Messages{
						Msgs: res,
					},
				},
			}
		case "send":
			mess := &models.Message{}
			err := mess.UnmarshalJSON(m.Payload)
			if err != nil {
				logger.Infof("Bad payload: %v", *m)
				chat.Send <- &ProcessWSMessage{
					From: u,
					WSM: &SendWSMessage{
						Action:  "error",
						Payload: "bad payload",
					},
				}
				return
			}
			logger.Infof("The message is: %v", *mess)

			if !u.Anon {
				mess.Author = new(uint)
				*mess.Author = u.UID
			}
			mess, err = database.CreateMessage(mess)
			if err != nil {
				logger.Infof("Message cannot be saved: %v", err)
				continue
			}
			logger.Infof("Message saved to database: %v", *m)
			chat.Send <- &ProcessWSMessage{
				From: u,
				WSM: &SendWSMessage{
					Action:  "send",
					Payload: mess,
				},
			}
		default:
			logger.Infof("Unknown WSMessage: %v", *m)

			chat.Send <- &ProcessWSMessage{
				From: u,
				WSM: &SendWSMessage{
					Action:  "error",
					Payload: "unknown action type",
				},
			}
		}
	}
}
